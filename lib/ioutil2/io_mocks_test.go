// Copyright (c) 2016 Yandex LLC. All rights reserved.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package ioutil2

import "io"

// File for generation mocks from io package.
// NOTE: remove _test filename suffix before run mockery.

//go:generate mockery -name=Closer -case=underscore -outpkg=iomock

type Closer interface {
	io.Closer
}

//go:generate mockery -name=Writer -case=underscore -outpkg=iomock

type Writer interface {
	io.Writer
}

//go:generate mockery -name=Reader -case=underscore -outpkg=iomock

type Reader interface {
	io.Writer
}

//go:generate mockery -name=ReadCloser -case=underscore -outpkg=iomock

type ReadCloser interface {
	io.ReadCloser
}

//go:generate mockery -name=WriteCloser -case=underscore -outpkg=iomock

type WriteCloser interface {
	io.WriteCloser
}
