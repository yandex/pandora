package spdy2

import (
	"runtime"

	"github.com/SlyMarbo/spdy/common"
	"github.com/SlyMarbo/spdy/spdy2/frames"
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
		if c.numBenignErrors > common.MaxBenignErrors && common.MaxBenignErrors > 0 {
			log.Println("Warning: Too many invalid stream IDs received. Ending connection.")
			c.protocolError(0)
			return
		}

		// ReadFrame takes care of the frame parsing for us.
		c.refreshReadTimeout()
		frame, err := frames.ReadFrame(c.buf)
		if err != nil {
			c.handleReadWriteError(err)
			return
		}

		// Print frame type.
		debug.Printf("Receiving %s:\n", frame.Name())

		// Decompress the frame's headers, if there are any.
		err = frame.Decompress(c.decompressor)
		if err != nil {
			log.Printf("Error in decompression: %v (%T).\n", err, frame)
			c.protocolError(0)
			return
		}

		// Print frame once the content's been decompressed.
		debug.Println(frame)

		// This is the main frame handling.
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

	// Enter the processing loop.
	i := 1
	for {

		// Once per 5 frames, pick randomly.
		var frame common.Frame
		if i == 0 { // Ignore priority.
			frame = c.selectFrameToSend(false)
		} else { // Normal selection.
			frame = c.selectFrameToSend(true)
		}

		i++
		if i >= 5 {
			i = 0
		}

		if frame == nil {
			c.Close()
			return
		}

		// Compress any name/value header blocks.
		err := frame.Compress(c.compressor)
		if err != nil {
			log.Printf("Error in compression: %v (%T).\n", err, frame)
			return
		}

		debug.Printf("Sending %s:\n", frame.Name())
		debug.Println(frame)

		// Leave the specifics of writing to the
		// connection up to the frame.
		c.refreshWriteTimeout()
		_, err = frame.WriteTo(c.conn)
		if err != nil {
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

	// Try in priority order first.
	if prioritise {
		for i := 0; i < 8; i++ {
			select {
			case frame = <-c.output[i]:
				return frame
			default:
			}
		}

		// No frames are immediately pending, so if the
		// connection is being closed, cease sending
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
