// Copyright 2013-14, Amahi. All rights reserved.
// Use of this source code is governed by the
// license that can be found in the LICENSE file.

// Frame related functions for generic for control/data frames
// as well as specific frame related functions

package spdy

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"strings"
)

// ========================================
// Control Frames
// ========================================
func (f controlFrame) Flags() frameFlags { return f.flags }
func (f controlFrame) Data() []byte      { return f.data }

func (f controlFrame) String() string {
	s := fmt.Sprintf("Control frame %s, flags: %s, size: %d\n", f.kind, f.flags, len(f.data))
	return s
}

func (f controlFrame) isFIN() bool { return f.flags&FLAG_FIN == FLAG_FIN }

func (f frameFlags) String() string {
	if f == FLAG_NONE {
		return "-"
	}
	if f == FLAG_FIN {
		return "FIN"
	}
	return fmt.Sprintf("0x%02x", int(f))
}

func (f controlFrameKind) String() string {
	switch f {
	case FRAME_SYN_STREAM:
		return "SYN_STREAM"
	case FRAME_SYN_REPLY:
		return "SYN_REPLY"
	case FRAME_RST_STREAM:
		return "RST_STREAM"
	case FRAME_SETTINGS:
		return "SETTINGS"
	case FRAME_PING:
		return "PING"
	case FRAME_GOAWAY:
		return "GOAWAY"
	case FRAME_HEADERS:
		return "HEADERS"
	case FRAME_WINDOW_UPDATE:
		return "WINDOW_UPDATE"
	}
	return fmt.Sprintf("Type(%#04x)", uint16(f))
}

func (f controlFrame) Write(w io.Writer) (n int64, err error) {

	total := len(f.data) + 8
	debug.Printf("Writing control frame %s, flags: %s, payload: %d", f.kind, f.flags, len(f.data))
	nn, err := writeFrame(w, []interface{}{uint16(0x8000) | uint16(0x0003), f.kind, f.flags}, f.data)
	if nn != total {
		log.Println("WARNING in controlFrame.Write, wrote", nn, "vs. frame size", total)
	}
	return int64(nn), err
}

// read the stream ID from the payload
// applies to SYN_STREAM, SYN_REPLY, RST_STREAM, GOAWAY, HEADERS and WINDOWS_UPDATE
func (f *controlFrame) streamID() (id streamID) {
	data := bytes.NewBuffer(f.data[0:4])
	err := binary.Read(data, binary.BigEndian, &id)
	if err != nil {
		log.Println("ERROR: Cannot read stream ID from a control frame that is supposed to have a stream ID:", err)
		id = 0
		return
	}
	id &= 0x7fffffff

	return
}

// ========================================
// Data Frames
// ========================================
func (f dataFrame) Flags() frameFlags { return f.flags }
func (f dataFrame) Data() []byte      { return f.data }
func (f dataFrame) isFIN() bool       { return f.flags&FLAG_FIN == FLAG_FIN }

func (f dataFrame) Write(w io.Writer) (n int64, err error) {
	total := len(f.data) + 8
	debug.Printf("Writing data frame, flags: %s, size: %d", f.flags, len(f.data))
	nn, err := writeFrame(w, []interface{}{f.stream & 0x7fffffff, f.flags}, f.data)
	if nn != total {
		log.Println("WARNING in dataFrame.Write, wrote", nn, "vs. frame size", total)
	}
	return int64(nn), err
}

func (f dataFrame) String() string {
	l := len(f.data)
	s := fmt.Sprintf("\n\tFrame: DATA of size %d, for stream #%d", l, f.stream)
	s += fmt.Sprintf(", Flags: %s", f.flags)
	if l > 16 {
		s += fmt.Sprintf("\n\tData: [%x .. %x]", f.data[0:6], f.data[l-6:])
	} else if l > 0 {
		s += fmt.Sprintf("\n\tData: [%x]", f.data)
	}
	return s
}

// ========================================
// Generic frame-writing utilities
// ========================================

func writeFrame(w io.Writer, head []interface{}, data []byte) (n int, err error) {
	var nn int
	// Header (40 bits)
	err = writeBinary(w, head...)
	if err != nil {
		return
	}
	n += 5 // frame head, in bytes, without the length field

	// Length (24 bits)
	length := len(data)
	nn, err = w.Write([]byte{
		byte(length & 0x00ff0000 >> 16),
		byte(length & 0x0000ff00 >> 8),
		byte(length & 0x000000ff),
	})
	n += nn
	if err != nil {
		log.Println("Write of length failed:", err)
		return
	}
	// Data
	if length > 0 {
		nn, err = w.Write(data)
		if err != nil {
			log.Println("Write of data failed:", err)
			return
		}
		n += nn
	}
	return
}

func writeBinary(r io.Writer, args ...interface{}) (err error) {
	for _, a := range args {
		err = binary.Write(r, binary.BigEndian, a)
		if err != nil {
			return
		}
	}
	return
}

// ========================================
// Generic frame-reading utilities
// ========================================

// readFrame reads an entire frame into memory
func readFrame(r io.Reader) (f frame, err error) {
	headBuffer := new(bytes.Buffer)
	_, err = io.CopyN(headBuffer, r, 5)
	if err != nil {
		return
	}
	if headBuffer.Bytes()[0]&0x80 == 0 {
		// Data
		df := dataFrame{}
		err = readBinary(headBuffer, &df.stream, &df.flags)
		if err != nil {
			return
		}
		df.data, err = readData(r)
		f = df
	} else {
		// Control
		cf := controlFrame{}
		headBuffer.ReadByte() // FIXME skip version word
		headBuffer.ReadByte()
		err = readBinary(headBuffer, &cf.kind, &cf.flags)
		if err != nil {
			return
		}
		cf.data, err = readData(r)
		f = cf
	}
	return
}

func readBinary(r io.Reader, args ...interface{}) (err error) {
	for _, a := range args {
		err = binary.Read(r, binary.BigEndian, a)
		if err != nil {
			return
		}
	}
	return
}

func readData(r io.Reader) (data []byte, err error) {
	lengthField := make([]byte, 3)
	_, err = io.ReadFull(r, lengthField)
	if err != nil {
		return
	}
	var length uint32
	length |= uint32(lengthField[0]) << 16
	length |= uint32(lengthField[1]) << 8
	length |= uint32(lengthField[2])

	if length > 0 {
		data = make([]byte, int(length))
		_, err = io.ReadFull(r, data)
		if err != nil {
			data = nil
			return
		}
	} else {
		data = []byte{}
	}
	return
}

// ========================================
// SYN_STREAM frame
// ========================================

func (frame frameSynStream) Flags() frameFlags {
	return frame.flags
}

func (frame frameSynStream) Data() []byte {
	buf := new(bytes.Buffer)
	// stream-id
	binary.Write(buf, binary.BigEndian, frame.stream&0x7fffffff)
	// associated-to-stream-id FIXME in the long term
	binary.Write(buf, binary.BigEndian, frame.associated_stream&0x7fffffff)
	// Priority & unused/reserved
	var misc uint16 = uint16((frame.priority & 0x7) << 13)
	binary.Write(buf, binary.BigEndian, misc)
	// debug.Println("Before header:", buf.Bytes())
	frame.session.headerWriter.writeHeader(buf, frame.header)
	// debug.Println("Compressed header:", buf.Bytes())
	return buf.Bytes()
}

func (frame frameSynStream) Write(w io.Writer) (n int64, err error) {
	f := controlFrame{kind: FRAME_SYN_STREAM, flags: frame.flags, data: frame.Data()}
	return f.Write(w)
}

// print details of the frame to a string
func (frame frameSynStream) String() string {
	s := fmt.Sprintf("\n\tFrame: SYN_STREAM, Stream #%d", frame.stream)
	s += fmt.Sprintf(", Flags: %s", frame.flags)
	s += fmt.Sprintf("\n\tHeaders:\n")
	for i := range frame.header {
		s += fmt.Sprintf("\t\t%s: %s\n", i, strings.Join(frame.header[i], ", "))
	}
	return s
}

// ========================================
// SYN_REPLY frame
// ========================================

func (frame frameSynReply) Flags() frameFlags {
	return frame.flags
}

func (frame frameSynReply) Data() []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, frame.stream&0x7fffffff)
	frame.session.headerWriter.writeHeader(buf, frame.headers)
	return buf.Bytes()
}

func (frame frameSynReply) Write(w io.Writer) (n int64, err error) {
	cf := controlFrame{kind: FRAME_SYN_REPLY, data: frame.Data()}
	return cf.Write(w)
}

// print details of the frame to a string
func (frame frameSynReply) String() string {
	s := fmt.Sprintf("\n\tFrame: SYN_REPLY, Stream #%d", frame.stream)
	s += fmt.Sprintf(", Flags: %s", frame.flags)
	s += fmt.Sprintf("\n\tHeaders:\n")
	for i := range frame.headers {
		s += fmt.Sprintf("\t\t%s: %s\n", i, strings.Join(frame.headers[i], ", "))
	}
	return s
}

// ========================================
// SETTINGS frame
// ========================================

func (s settings) Flags() frameFlags {
	return s.flags
}

func (s settings) Data() []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, s.count)
	for i := range s.svp {
		binary.Write(buf, binary.BigEndian, s.svp[i].flags)
		binary.Write(buf, binary.BigEndian, s.svp[i].id)
		binary.Write(buf, binary.BigEndian, s.svp[i].value)
	}
	return buf.Bytes()
}

func (s settings) Write(w io.Writer) (n int64, err error) {
	cf := controlFrame{kind: FRAME_SETTINGS, flags: s.flags, data: s.Data()}
	return cf.Write(w)
}

func (s settings) String() (r string) {
	r = fmt.Sprintf("\n\tFrame: SETTINGS, Flags: %s\n", s.flags)
	r += fmt.Sprintf("\tValue/Pairs (%d):\n", s.count)
	for i := range s.svp {
		r += fmt.Sprintf("\t\t%d: %d\tflags: %d\n", s.svp[i].id, s.svp[i].value, s.svp[i].flags)
	}
	return
}

// ========================================
// WINDOW_UPDATE frame
// ========================================

// takes a frame and the delta window size and returns a WINDOW_UPDATE frame
func windowUpdateFor(id streamID, dws int) frame {

	data := new(bytes.Buffer)
	binary.Write(data, binary.BigEndian, id)
	binary.Write(data, binary.BigEndian, uint32(dws))

	return controlFrame{kind: FRAME_WINDOW_UPDATE, data: data.Bytes()}
}
