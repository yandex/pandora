package spdy3

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/SlyMarbo/spdy/common"
	"github.com/SlyMarbo/spdy/spdy3/frames"
)

// Request is used to make a client request.
func (c *Conn) Request(request *http.Request, receiver common.Receiver, priority common.Priority) (common.Stream, error) {
	if c.Closed() {
		return nil, common.ErrConnClosed
	}
	c.goawayLock.Lock()
	goaway := c.goawayReceived || c.goawaySent
	c.goawayLock.Unlock()
	if goaway {
		return nil, common.ErrGoaway
	}

	if c.server != nil {
		return nil, errors.New("Error: Only clients can send requests.")
	}

	// Check stream limit would allow the new stream.
	if !c.requestStreamLimit.Add() {
		return nil, errors.New("Error: Max concurrent streams limit exceeded.")
	}

	if !priority.Valid(3) {
		return nil, errors.New("Error: Priority must be in the range 0 - 7.")
	}

	url := request.URL
	if url == nil || url.Scheme == "" || url.Host == "" {
		return nil, errors.New("Error: Incomplete path provided to resource.")
	}

	// Prepare the SYN_STREAM.
	path := url.Path
	if url.RawQuery != "" {
		path += "?" + url.RawQuery
	}
	if url.Fragment != "" {
		path += "#" + url.Fragment
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	host := url.Host
	if len(request.Host) > 0 {
		host = request.Host
	}
	syn := new(frames.SYN_STREAM)
	syn.Priority = priority
	syn.Header = request.Header
	syn.Header.Set(":method", request.Method)
	syn.Header.Set(":path", path)
	syn.Header.Set(":version", "HTTP/1.1")
	syn.Header.Set(":host", host)
	syn.Header.Set(":scheme", url.Scheme)

	// Prepare the request body, if any.
	body := make([]*frames.DATA, 0, 1)
	if request.Body != nil {
		buf := make([]byte, 32*1024)
		n, err := request.Body.Read(buf)
		if err != nil && err != io.EOF {
			return nil, err
		}
		total := n
		for n > 0 {
			data := new(frames.DATA)
			data.Data = make([]byte, n)
			copy(data.Data, buf[:n])
			body = append(body, data)
			n, err = request.Body.Read(buf)
			if err != nil && err != io.EOF {
				return nil, err
			}
			total += n
		}

		// Half-close the stream.
		if len(body) == 0 {
			syn.Flags = common.FLAG_FIN
		} else {
			syn.Header.Set("Content-Length", fmt.Sprint(total))
			body[len(body)-1].Flags = common.FLAG_FIN
		}
		request.Body.Close()
	} else {
		syn.Flags = common.FLAG_FIN
	}

	// Send.
	c.streamCreation.Lock()
	defer c.streamCreation.Unlock()

	c.lastRequestStreamIDLock.Lock()
	if c.lastRequestStreamID == 0 {
		c.lastRequestStreamID = 1
	} else {
		c.lastRequestStreamID += 2
	}
	syn.StreamID = c.lastRequestStreamID
	c.lastRequestStreamIDLock.Unlock()
	if syn.StreamID > common.MAX_STREAM_ID {
		return nil, errors.New("Error: All client streams exhausted.")
	}
	c.output[0] <- syn
	for _, frame := range body {
		frame.StreamID = syn.StreamID
		c.output[0] <- frame
	}

	// Create the request stream.
	out := NewRequestStream(c, syn.StreamID, c.output[0])
	out.Request = request
	out.Receiver = receiver
	out.AddFlowControl(c.flowControl)
	c.streamsLock.Lock()
	c.streams[syn.StreamID] = out // Store in the connection map.
	c.streamsLock.Unlock()

	return out, nil
}

func (c *Conn) RequestResponse(request *http.Request, receiver common.Receiver, priority common.Priority) (*http.Response, error) {
	res := common.NewResponse(request, receiver)

	// Send the request.
	stream, err := c.Request(request, res, priority)
	if err != nil {
		return nil, err
	}

	// Let the request run its course.
	stream.Run()

	return res.Response(), c.shutdownError
}
