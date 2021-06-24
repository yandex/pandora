package errutil

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJoin(t *testing.T) {
	err1 := errors.New("first error")
	err2 := errors.New("second error")

	var err error

	err = Join(err1, nil)
	assert.Equal(t, err, err1)

	err = Join(nil, err2)
	assert.Equal(t, err, err2)

	err = Join(err1, err2)

	assert.NotNil(t, err)

	assert.True(t, strings.Contains(err.Error(), err1.Error()))
	assert.True(t, strings.Contains(err.Error(), err2.Error()))

}
