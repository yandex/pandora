package spdy2

import (
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/SlyMarbo/spdy/common"
	"github.com/SlyMarbo/spdy/spdy2/frames"
)

// Ping is used by spdy.PingServer and spdy.PingClient to send
// SPDY PINGs.
func (c *Conn) Ping() (<-chan bool, error) {
	if c.Closed() {
		return nil, errors.New("Error: Conn has been closed.")
	}

	ping := new(frames.PING)
	c.nextPingIDLock.Lock()
	pid := c.nextPingID
	if pid+2 < pid {
		if pid&1 == 0 {
			c.nextPingID = 2
		} else {
			c.nextPingID = 1
		}
	} else {
		c.nextPingID += 2
	}
	c.nextPingIDLock.Unlock()
	ping.PingID = pid
	c.output[0] <- ping
	ch := make(chan bool, 1)
	c.pingsLock.Lock()
	c.pings[pid] = ch
	c.pingsLock.Unlock()

	return ch, nil
}

// Push is used to issue a server push to the client. Note that this cannot be performed
// by clients.
func (c *Conn) Push(resource string, origin common.Stream) (common.PushStream, error) {
	c.goawayLock.Lock()
	goaway := c.goawayReceived || c.goawaySent
	c.goawayLock.Unlock()
	if goaway {
		return nil, common.ErrGoaway
	}

	if c.server == nil {
		return nil, errors.New("Error: Only servers can send pushes.")
	}

	// Parse and check URL.
	url, err := url.Parse(resource)
	if err != nil {
		return nil, err
	}
	if url.Scheme == "" || url.Host == "" {
		return nil, errors.New("Error: Incomplete path provided to resource.")
	}
	resource = url.String()

	// Ensure the resource hasn't been pushed on the given stream already.
	if c.pushedResources[origin] == nil {
		c.pushedResources[origin] = map[string]struct{}{
			resource: struct{}{},
		}
	} else if _, ok := c.pushedResources[origin][url.String()]; !ok {
		c.pushedResources[origin][resource] = struct{}{}
	} else {
		return nil, errors.New("Error: Resource already pushed to this stream.")
	}

	// Check stream limit would allow the new stream.
	if !c.pushStreamLimit.Add() {
		return nil, errors.New("Error: Max concurrent streams limit exceeded.")
	}

	// Verify that path is prefixed with / as required by spec.
	path := url.Path
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	// Prepare the SYN_STREAM.
	push := new(frames.SYN_STREAM)
	push.Flags = common.FLAG_UNIDIRECTIONAL
	push.AssocStreamID = origin.StreamID()
	push.Priority = 3
	push.Header = make(http.Header)
	push.Header.Set("scheme", url.Scheme)
	push.Header.Set("host", url.Host)
	push.Header.Set("url", path)
	push.Header.Set("version", "HTTP/1.1")

	// Send.
	c.streamCreation.Lock()
	defer c.streamCreation.Unlock()

	c.lastPushStreamIDLock.Lock()
	c.lastPushStreamID += 2
	newID := c.lastPushStreamID
	c.lastPushStreamIDLock.Unlock()
	if newID > common.MAX_STREAM_ID {
		return nil, errors.New("Error: All server streams exhausted.")
	}
	push.StreamID = newID
	c.output[0] <- push

	// Create the PushStream.
	out := NewPushStream(c, newID, origin, c.output[3])

	// Store in the connection map.
	c.streamsLock.Lock()
	c.streams[newID] = out
	c.streamsLock.Unlock()

	return out, nil
}
