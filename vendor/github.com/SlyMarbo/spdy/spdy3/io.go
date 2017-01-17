package spdy3

import (
	"runtime"

	"github.com/SlyMarbo/spdy/common"
	"github.com/SlyMarbo/spdy/spdy3/frames"
)

// readFrames is the main processing loop, where frames
// are read from the connection and processed individually.
// Returning from readFrames begins the cleanup and exit
// process for this connection.
func (c *Conn) readFrames() {
	// Ensure no panics happen.
	defer func() {
		if v := recover(); v != nil {
			if !c.Closed() {
				log.Printf("Encountered receive error: %v (%[1]T)\n", v)
			}
		}
	}()

	for {
		// This is the mechanism for handling too many benign errors.
		// By default MaxBenignErrors is 0, which ignores errors.
		too_many := c.numBenignErrors > common.MaxBenignErrors && common.MaxBenignErrors > 0
		if c.criticalCheck(too_many, 0, "Ending connection for benign error buildup") {
			return
		}

		// ReadFrame takes care of the frame parsing for us.
		c.refreshReadTimeout()
		frame, err := frames.ReadFrame(c.buf, c.Subversion)
		if err != nil {
			c.handleReadWriteError(err)
			return
		}

		debug.Printf("Receiving %s:\n", frame.Name()) // Print frame type.

		// Decompress the frame's headers, if there are any.
		err = frame.Decompress(c.decompressor)
		if c.criticalCheck(err != nil, 0, "Decompression: %v", err) {
			return
		}

		debug.Println(frame) // Print frame once the content's been decompressed.

		if c.processFrame(frame) {
			return
		}
	}
}

// send is run in a separate goroutine. It's used
// to ensure clear interleaving of frames and to
// provide assurances of priority and structure.
func (c *Conn) send() {
	// Catch any panics.
	defer func() {
		if v := recover(); v != nil {
			if !c.Closed() {
				log.Printf("Encountered send error: %v (%[1]T)\n", v)
			}
		}
	}()

	for i := 1; ; i++ {
		if i >= 5 {
			i = 0 // Once per 5 frames, pick randomly.
		}

		var frame common.Frame
		if i == 0 { // Ignore priority.
			frame = c.selectFrameToSend(false)
		} else { // Normal selection.
			frame = c.selectFrameToSend(true)
		}

		if frame == nil {
			c.Close()
			return
		}

		// Process connection-level flow control.
		if c.Subversion > 0 {
			c.connectionWindowLock.Lock()
			if frame, ok := frame.(*frames.DATA); ok {
				size := int64(len(frame.Data))
				constrained := false
				sending := size
				if sending > c.connectionWindowSize {
					sending = c.connectionWindowSize
					constrained = true
				}
				if sending < 0 {
					sending = 0
				}

				c.connectionWindowSize -= sending

				if constrained {
					// Chop off what we can send now.
					partial := new(frames.DATA)
					partial.Flags = frame.Flags
					partial.StreamID = frame.StreamID
					partial.Data = make([]byte, int(sending))
					copy(partial.Data, frame.Data[:sending])
					frame.Data = frame.Data[sending:]

					// Buffer this frame and try again.
					if c.dataBuffer == nil {
						c.dataBuffer = []*frames.DATA{frame}
					} else {
						buffer := make([]*frames.DATA, 1, len(c.dataBuffer)+1)
						buffer[0] = frame
						buffer = append(buffer, c.dataBuffer...)
						c.dataBuffer = buffer
					}

					frame = partial
				}
			}
			c.connectionWindowLock.Unlock()
		}

		// Compress any name/value header blocks.
		err := frame.Compress(c.compressor)
		if err != nil {
			log.Printf("Error in compression: %v (type %T).\n", err, frame)
			c.Close()
			return
		}

		debug.Printf("Sending %s:\n", frame.Name())
		debug.Println(frame)

		// Leave the specifics of writing to the
		// connection up to the frame.
		c.refreshWriteTimeout()
		if _, err = frame.WriteTo(c.conn); err != nil {
			c.handleReadWriteError(err)
			return
		}
	}
}

// selectFrameToSend follows the specification's guidance
// on frame priority, sending frames with higher priority
// (a smaller number) first. If the given boolean is false,
// this priority is temporarily ignored, which can be used
// when high load is ignoring low-priority frames.
func (c *Conn) selectFrameToSend(prioritise bool) (frame common.Frame) {
	if c.Closed() {
		return nil
	}

	// Try buffered DATA frames first.
	if c.Subversion > 0 {
		if c.dataBuffer != nil {
			if len(c.dataBuffer) == 0 {
				c.dataBuffer = nil
			} else {
				first := c.dataBuffer[0]
				if c.connectionWindowSize >= int64(8+len(first.Data)) {
					if len(c.dataBuffer) > 1 {
						c.dataBuffer = c.dataBuffer[1:]
					} else {
						c.dataBuffer = nil
					}
					return first
				}
			}
		}
	}

	// Then in priority order.
	if prioritise {
		for i := 0; i < 8; i++ {
			select {
			case frame = <-c.output[i]:
				return frame
			default:
			}
		}

		// No frames are immediately pending, so if the
		// cection is being closed, cease sending
		// safely.
		c.sendingLock.Lock()
		if c.sending != nil {
			close(c.sending)
			c.sendingLock.Unlock()
			runtime.Goexit()
		}
		c.sendingLock.Unlock()
	}

	// Wait for any frame.
	select {
	case frame = <-c.output[0]:
		return frame
	case frame = <-c.output[1]:
		return frame
	case frame = <-c.output[2]:
		return frame
	case frame = <-c.output[3]:
		return frame
	case frame = <-c.output[4]:
		return frame
	case frame = <-c.output[5]:
		return frame
	case frame = <-c.output[6]:
		return frame
	case frame = <-c.output[7]:
		return frame
	case _ = <-c.stop:
		return nil
	}
}
