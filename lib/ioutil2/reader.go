package ioutil2

import "io"

type ReaderWrapper interface {
	Unwrap() io.Reader
}

// TODO(skipor): test

func NewMultiPassReader(r io.Reader, passes int) io.Reader {
	if passes == 1 {
		return r
	}
	rs, isSeakable := r.(io.ReadSeeker)
	if !isSeakable {
		return r
	}
	return &MultiPassReader{rs: rs, passesLimit: passes}
}

type MultiPassReader struct {
	rs          io.ReadSeeker
	passesCount int
	passesLimit int
}

func (r *MultiPassReader) Read(p []byte) (n int, err error) {
	n, err = r.rs.Read(p)
	if err == io.EOF {
		r.passesCount++
		if r.passesLimit <= 0 || r.passesCount < r.passesLimit {
			_, err = r.rs.Seek(0, io.SeekStart)
		}
	}
	return
}

// func (r *MultiPassReader) PassesLeft() int {
//	return r.PassesLeft()
// }

func (r *MultiPassReader) Unwrap() io.Reader {
	return r.rs
}
