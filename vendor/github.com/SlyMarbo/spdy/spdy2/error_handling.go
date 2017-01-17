package spdy2

import (
	"io"
	"net"
	"time"

	"github.com/SlyMarbo/spdy/common"
	"github.com/SlyMarbo/spdy/spdy2/frames"
)

// check returns the error condition and
// updates the connection accordingly.
func (c *Conn) check(condition bool, format string, v ...interface{}) bool {
	if !condition {
		return false
	}
	log.Printf("Error: "+format+".\n", v...)
	c.numBenignErrors++
	return true
}

// criticalCheck returns the error condition
// and ends the connection accordingly.
func (c *Conn) criticalCheck(condition bool, sid common.StreamID, format string, v ...interface{}) bool {
	if !condition {
		return false
	}
	log.Printf("Error: "+format+".\n", v...)
	c.protocolError(sid)
	return true
}

func (c *Conn) _RST_STREAM(streamID common.StreamID, status common.StatusCode) {
	rst := new(frames.RST_STREAM)
	rst.StreamID = streamID
	rst.Status = status
	c.output[0] <- rst
}

func (c *Conn) _GOAWAY() {
	c.output[0] <- new(frames.GOAWAY)
	c.Close()
}

// handleReadWriteError differentiates between normal and
// unexpected errors when performing I/O with the network,
// then shuts down the connection.
func (c *Conn) handleReadWriteError(err error) {
	if _, ok := err.(*net.OpError); ok || err == io.EOF || err == common.ErrConnNil ||
		err.Error() == "use of closed network connection" {
		// Server has closed the TCP connection.
		debug.Println("Note: Endpoint has discected.")
	} else {
		// Unexpected error which prevented a read/write.
		log.Printf("Error: Encountered error: %q (%T)\n", err.Error(), err)
	}

	// Make sure c.Close succeeds and sending stops.
	c.sendingLock.Lock()
	if c.sending == nil {
		c.sending = make(chan struct{})
	}
	c.sendingLock.Unlock()

	c.Close()
}

// protocolError informs the other endpoint that a protocol error has
// occurred, stops all running streams, and ends the connection.
func (c *Conn) protocolError(streamID common.StreamID) {
	reply := new(frames.RST_STREAM)
	reply.StreamID = streamID
	reply.Status = common.RST_STREAM_PROTOCOL_ERROR
	select {
	case c.output[0] <- reply:
	case <-time.After(100 * time.Millisecond):
		debug.Println("Failed to send PROTOCOL_ERROR RST_STREAM.")
	}
	c.shutdownError = reply
	c.Close()
}
