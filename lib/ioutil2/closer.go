package ioutil2

// NopCloser may be embedded to any struct to implement io.Closer doing nothing on closer.
type NopCloser struct{}

func (NopCloser) Close() error { return nil }
