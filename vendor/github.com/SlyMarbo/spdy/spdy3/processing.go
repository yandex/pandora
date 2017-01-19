package spdy3

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/SlyMarbo/spdy/common"
	"github.com/SlyMarbo/spdy/spdy3/frames"
)

// processFrame handles the initial processing of the given
// frame, before passing it on to the relevant helper func,
// if necessary. The returned boolean indicates whether the
// connection is closing.
func (c *Conn) processFrame(frame common.Frame) bool {
	switch frame := frame.(type) {

	case *frames.SYN_STREAM:
		if c.server == nil {
			c.handlePush(frame)
		} else {
			c.handleRequest(frame)
		}
	case *frames.SYN_STREAMV3_1:
		f3 := new(frames.SYN_STREAM)
		f3.Flags = frame.Flags
		f3.StreamID = frame.StreamID
		f3.AssocStreamID = frame.AssocStreamID
		f3.Priority = frame.Priority
		f3.Slot = 0
		f3.Header = frame.Header
		if c.server == nil {
			c.handlePush(f3)
		} else {
			c.handleRequest(f3)
		}

	case *frames.SYN_REPLY:
		c.handleSynReply(frame)

	case *frames.RST_STREAM:
		if frame.Status.IsFatal() {
			code := frame.Status.String()
			log.Printf("Warning: Received %s on stream %d. Closing connection.\n", code, frame.StreamID)
			c.shutdownError = frame
			c.Close()
			return true
		}
		c.handleRstStream(frame)

	case *frames.SETTINGS:
		for _, setting := range frame.Settings {
			c.receivedSettings[setting.ID] = setting
			switch setting.ID {
			case common.SETTINGS_INITIAL_WINDOW_SIZE:
				c.initialWindowSizeLock.Lock()
				initial := int64(c.initialWindowSize)
				current := c.connectionWindowSize
				inbound := int64(setting.Value)
				if initial != inbound {
					if initial > inbound {
						c.connectionWindowSize = inbound - (initial - current)
					} else {
						c.connectionWindowSize += (inbound - initial)
					}
					c.initialWindowSize = setting.Value
				}
				c.initialWindowSizeLock.Unlock()

			case common.SETTINGS_MAX_CONCURRENT_STREAMS:
				if c.server == nil {
					c.requestStreamLimit.SetLimit(setting.Value)
				} else {
					c.pushStreamLimit.SetLimit(setting.Value)
				}
			}
		}

	case *frames.PING:
		// Check whether Ping ID is a response.
		c.nextPingIDLock.Lock()
		next := c.nextPingID
		c.nextPingIDLock.Unlock()
		if frame.PingID&1 == next&1 {
			c.pingsLock.Lock()
			if c.check(c.pings[frame.PingID] == nil, "Ignored unrequested PING %d", frame.PingID) {
				c.pingsLock.Unlock()
				return false
			}
			c.pings[frame.PingID] <- true
			close(c.pings[frame.PingID])
			delete(c.pings, frame.PingID)
			c.pingsLock.Unlock()
		} else {
			debug.Println("Received PING. Replying...")
			c.output[0] <- frame
		}

	case *frames.GOAWAY:
		lastProcessed := frame.LastGoodStreamID
		c.streamsLock.Lock()
		for streamID, stream := range c.streams {
			if streamID&1 == c.oddity && streamID > lastProcessed {
				// Stream is locally-sent and has not been processed.
				// TODO: Inform the server that the push has not been successful.
				stream.Close()
			}
		}
		c.streamsLock.Unlock()
		c.shutdownError = frame
		c.goawayLock.Lock()
		c.goawayReceived = true
		c.goawayLock.Unlock()

	case *frames.HEADERS:
		c.handleHeaders(frame)

	case *frames.WINDOW_UPDATE:
		c.handleWindowUpdate(frame)

	case *frames.CREDENTIAL:
		if c.Subversion > 0 {
			return false
		}
		if c.server == nil || c.certificates == nil {
			log.Println("Ignored unexpected CREDENTIAL.")
			return false
		}
		if frame.Slot >= c.vectorIndex {
			setting := new(frames.SETTINGS)
			setting.Settings = common.Settings{
				common.SETTINGS_CLIENT_CERTIFICATE_VECTOR_SIZE: &common.Setting{
					ID:    common.SETTINGS_CLIENT_CERTIFICATE_VECTOR_SIZE,
					Value: uint32(frame.Slot + 4),
				},
			}
			c.output[0] <- setting
			c.vectorIndex += 4
		}
		c.certificates[frame.Slot] = frame.Certificates

	case *frames.DATA:
		if c.Subversion > 0 {
			// The transfer window shouldn't already be negative.
			if c.connectionWindowSizeThere < 0 {
				c._GOAWAY(common.GOAWAY_FLOW_CONTROL_ERROR)
				return false
			}

			c.connectionWindowSizeThere -= int64(len(frame.Data))

			c.flowControlLock.Lock()
			f := c.flowControl
			c.flowControlLock.Unlock()
			delta := f.ReceiveData(0, c.initialWindowSizeThere, c.connectionWindowSizeThere)
			if delta != 0 {
				grow := new(frames.WINDOW_UPDATE)
				grow.StreamID = 0
				grow.DeltaWindowSize = delta
				c.output[0] <- grow
				c.connectionWindowSizeThere += int64(grow.DeltaWindowSize)
			}
		}
		if c.server == nil {
			c.handleServerData(frame)
		} else {
			c.handleClientData(frame)
		}

	default:
		log.Println(fmt.Sprintf("Ignored unexpected frame type %T", frame))
	}
	return false
}

// handleClientData performs the processing of DATA frames sent by the client.
func (c *Conn) handleClientData(frame *frames.DATA) {
	sid := frame.StreamID

	if c.check(c.server == nil, "Client received request") {
		return
	}

	// Handle request data.
	if c.check(sid&1 == 0, "Received DATA with even Stream ID %d", sid) {
		return
	}

	// Check stream ID is valid.
	if c.criticalCheck(!sid.Valid(), sid, "Received DATA with excessive Stream ID %d", sid) {
		return
	}

	// Check stream is open.
	c.streamsLock.Lock()
	stream := c.streams[sid]
	c.streamsLock.Unlock()
	closed := stream == nil || stream.State().ClosedThere()
	if c.check(closed, "Received DATA with unopened or closed Stream ID %d", sid) {
		return
	}

	stream.ReceiveFrame(frame) // Send data to stream.
}

// handleHeaders performs the processing of HEADERS frames.
func (c *Conn) handleHeaders(frame *frames.HEADERS) {
	sid := frame.StreamID

	// Handle push headers.
	if sid&1 == 0 && c.server == nil {
		// Ignore refused push headers.
		if req := c.pushRequests[sid]; req != nil && c.PushReceiver != nil {
			c.PushReceiver.ReceiveHeader(req, frame.Header)
		}
		return
	}

	// Check stream is open.
	c.streamsLock.Lock()
	stream := c.streams[sid]
	c.streamsLock.Unlock()
	closed := stream == nil || stream.State().ClosedThere()
	if c.check(closed, "Received HEADERS with unopened or closed Stream ID %d", sid) {
		return
	}

	stream.ReceiveFrame(frame) // Send headers to stream.
}

// handlePush performs the processing of SYN_STREAM frames forming a server push.
func (c *Conn) handlePush(frame *frames.SYN_STREAM) {
	sid := frame.StreamID

	// Check stream creation is allowed.
	c.goawayLock.Lock()
	goaway := c.goawayReceived || c.goawaySent
	c.goawayLock.Unlock()
	if goaway || c.Closed() {
		return
	}

	if c.server != nil {
		log.Println("Error: Only clients can receive server pushes.")
		return
	}

	if c.check(sid&1 != 0, "Received SYN_STREAM with odd Stream ID %d", sid) {
		return
	}

	c.lastPushStreamIDLock.Lock()
	lsid := c.lastPushStreamID
	c.lastPushStreamIDLock.Unlock()
	if c.check(sid <= lsid, "Received SYN_STREAM with Stream ID %d, less than %d", sid, lsid) {
		return
	}

	if c.criticalCheck(!sid.Valid(), sid, "Received SYN_STREAM with excessive Stream ID %d", sid) {
		return
	}

	// Stream ID is fine.

	// Check stream limit would allow the new stream.
	if !c.pushStreamLimit.Add() {
		c._RST_STREAM(sid, common.RST_STREAM_REFUSED_STREAM)
		return
	}

	ok := frame.Priority.Valid(3)
	if c.criticalCheck(!ok, sid, "Received SYN_STREAM with invalid priority %d", frame.Priority) {
		return
	}

	// Parse the request.
	header := frame.Header
	rawUrl := header.Get(":scheme") + "://" + header.Get(":host") + header.Get(":path")
	url, err := url.Parse(rawUrl)
	if c.check(err != nil, "Received SYN_STREAM with invalid request URL (%v)", err) {
		return
	}

	vers := header.Get(":version")
	major, minor, ok := http.ParseHTTPVersion(vers)
	if c.check(!ok, "Invalid HTTP version: "+vers) {
		return
	}

	method := header.Get(":method")
	request := &http.Request{
		Method:     method,
		URL:        url,
		Proto:      vers,
		ProtoMajor: major,
		ProtoMinor: minor,
		RemoteAddr: c.remoteAddr,
		Header:     header,
		Host:       url.Host,
		RequestURI: url.RequestURI(),
		TLS:        c.tlsState,
	}

	// Check whether the receiver wants this resource.
	if c.PushReceiver != nil && !c.PushReceiver.ReceiveRequest(request) {
		c._RST_STREAM(sid, common.RST_STREAM_REFUSED_STREAM)
		return
	}

	if c.PushReceiver != nil {
		c.pushRequests[sid] = request
		c.lastPushStreamIDLock.Lock()
		c.lastPushStreamID = sid
		c.lastPushStreamIDLock.Unlock()
		c.PushReceiver.ReceiveHeader(request, frame.Header)
	}
}

// handleRequest performs the processing of SYN_STREAM request frames.
func (c *Conn) handleRequest(frame *frames.SYN_STREAM) {
	// Check stream creation is allowed.
	c.goawayLock.Lock()
	goaway := c.goawayReceived || c.goawaySent
	c.goawayLock.Unlock()
	if goaway || c.Closed() {
		return
	}

	sid := frame.StreamID

	if c.criticalCheck(c.server == nil, sid, "Client received request") {
		return
	}

	if c.check(sid&1 == 0, "Received SYN_STREAM with even Stream ID %d", sid) {
		return
	}

	c.lastRequestStreamIDLock.Lock()
	lsid := c.lastRequestStreamID
	c.lastRequestStreamIDLock.Unlock()
	if c.check(sid <= lsid && lsid != 0, "Received SYN_STREAM with Stream ID %d, less than %d", sid, lsid) {
		return
	}

	if c.criticalCheck(!sid.Valid(), sid, "Received SYN_STREAM with excessive Stream ID %d", sid) {
		return
	}

	// Stream ID is fine.

	// Check stream limit would allow the new stream.
	if !c.requestStreamLimit.Add() {
		c._RST_STREAM(sid, common.RST_STREAM_REFUSED_STREAM)
		return
	}

	p := frame.Priority
	if c.criticalCheck(!p.Valid(3), sid, "Received SYN_STREAM with bad priority %d.\n", p) {
		return
	}

	// Create and start new stream.
	nextStream := c.newStream(frame)
	if nextStream == nil {
		return
	}

	c.streamsLock.Lock()
	c.streams[sid] = nextStream
	c.streamsLock.Unlock()
	c.lastRequestStreamIDLock.Lock()
	c.lastRequestStreamID = sid
	c.lastRequestStreamIDLock.Unlock()
	go nextStream.Run()
}

// handleRstStream performs the processing of RST_STREAM frames.
func (c *Conn) handleRstStream(frame *frames.RST_STREAM) {
	sid := frame.StreamID
	c.streamsLock.Lock()
	stream := c.streams[sid]
	c.streamsLock.Unlock()

	// Determine the status code and react accordingly.
	switch frame.Status {
	case common.RST_STREAM_INVALID_STREAM,
		common.RST_STREAM_STREAM_ALREADY_CLOSED:
		if stream != nil {
			go stream.Close()
		}
		fallthrough
	case common.RST_STREAM_STREAM_IN_USE:
		c.check(true, "Received %s for stream ID %d", frame.Status, sid)

	case common.RST_STREAM_CANCEL: // Allow cancelling of pushes.
		if c.check(sid&1 == c.oddity && stream == nil, "Cannot cancel locally-sent streams") {
			return
		}
		fallthrough
	case common.RST_STREAM_REFUSED_STREAM:
		if stream != nil {
			go stream.Close()
		}

	case common.RST_STREAM_FLOW_CONTROL_ERROR:
		c.numBenignErrors++

	case common.RST_STREAM_INVALID_CREDENTIALS:
		if c.Subversion > 0 {
			return
		}
		c.check(true, "Received INVALID_CREDENTIALS for stream ID %d.\n", sid)

	default:
		c.criticalCheck(true, sid, "Received unknown RST_STREAM status code %d", frame.Status)
	}
}

// handleServerData performs the processing of DATA frames sent by the server.
func (c *Conn) handleServerData(frame *frames.DATA) {
	sid := frame.StreamID

	if sid&1 == 0 { // Handle push data.
		// Ignore refused push data.
		if req := c.pushRequests[sid]; req != nil && c.PushReceiver != nil {
			c.PushReceiver.ReceiveData(req, frame.Data, frame.Flags.FIN())
		}
		return
	}

	// Check stream is open.
	c.streamsLock.Lock()
	stream := c.streams[sid]
	c.streamsLock.Unlock()
	closed := stream == nil || stream.State().ClosedThere()
	if c.check(closed, "Received DATA with unopened or closed Stream ID %d", sid) {
		return
	}

	// Stream ID is fine.
	stream.ReceiveFrame(frame)
}

// handleSynReply performs the processing of SYN_REPLY frames.
func (c *Conn) handleSynReply(frame *frames.SYN_REPLY) {
	sid := frame.StreamID

	if c.check(c.server != nil, "Only clients can receive SYN_REPLY frames") {
		return
	}

	if c.check(sid&1 == 0, "Received SYN_REPLY with even Stream ID %d", sid) {
		return
	}

	if c.criticalCheck(!sid.Valid(), sid, "Received SYN_REPLY with excessive Stream ID %d", sid) {
		return
	}

	c.streamsLock.Lock()
	stream := c.streams[sid]
	c.streamsLock.Unlock()
	closed := stream == nil || stream.State().ClosedThere()
	if c.check(closed, "Received SYN_REPLY with unopened or closed Stream ID %d", sid) {
		return
	}

	// Stream ID is fine.
	stream.ReceiveFrame(frame)
}

// handleWindowUpdate performs the processing of WINDOW_UPDATE frames.
func (c *Conn) handleWindowUpdate(frame *frames.WINDOW_UPDATE) {
	sid := frame.StreamID

	if c.criticalCheck(!sid.Valid(), sid, "Received WINDOW_UPDATE with excessive Stream ID %d", sid) {
		return
	}

	delta := frame.DeltaWindowSize
	bad := delta > common.MAX_DELTA_WINDOW_SIZE || delta < 1
	if c.criticalCheck(bad, sid, "Received WINDOW_UPDATE with invalid delta window size %d", delta) {
		return
	}

	// Handle connection-level flow control.
	if sid.Zero() && c.Subversion > 0 {
		c.connectionWindowLock.Lock()
		defer c.connectionWindowLock.Unlock()

		if int64(delta)+c.connectionWindowSize > common.MAX_TRANSFER_WINDOW_SIZE {
			goaway := new(frames.GOAWAY)
			if c.server != nil {
				c.lastRequestStreamIDLock.Lock()
				goaway.LastGoodStreamID = c.lastRequestStreamID
				c.lastRequestStreamIDLock.Unlock()
			} else {
				c.lastPushStreamIDLock.Lock()
				goaway.LastGoodStreamID = c.lastPushStreamID
				c.lastPushStreamIDLock.Unlock()
			}
			goaway.Status = common.GOAWAY_FLOW_CONTROL_ERROR
			c.output[0] <- goaway
			return
		}
		c.connectionWindowSize += int64(delta)
		return
	}

	// Check stream is open.
	c.streamsLock.Lock()
	stream := c.streams[sid]
	c.streamsLock.Unlock()
	if stream == nil || stream.State().ClosedHere() {
		// This is almost certainly benign
		return
	}

	// Stream ID is fine.
	stream.ReceiveFrame(frame)
}
