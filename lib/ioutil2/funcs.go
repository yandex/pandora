package ioutil2

type ReaderFunc func(p []byte) (n int, err error)
type WriterFunc func(p []byte) (n int, err error)
type CloserFunc func() error

func (f ReaderFunc) Read(p []byte) (int, error)  { return f(p) }
func (f WriterFunc) Write(p []byte) (int, error) { return f(p) }
func (f CloserFunc) Close() error                { return f() }

type StringerFunc func() string

func (f StringerFunc) String() string { return f() }
