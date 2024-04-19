package str

import (
	"errors"
	"math/rand"
	"strings"
	"time"
)

const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_-"

var randSource = rand.New(rand.NewSource(time.Now().UnixNano()))

func ParseStringFunc(shoot string) (string, []string, error) {
	openIdx := strings.IndexRune(shoot, '(')
	if openIdx == -1 {
		closeIdx := strings.IndexRune(shoot, ')')
		if closeIdx != -1 {
			return "", nil, errors.New("invalid close bracket position")
		}
		return shoot, nil, nil
	}
	name := strings.TrimSpace(shoot[:openIdx])

	arg := strings.TrimSpace(shoot[openIdx+1:])
	closeIdx := strings.IndexRune(arg, ')')
	if closeIdx != len(arg)-1 || closeIdx == -1 {
		return "", nil, errors.New("invalid close bracket position")
	}
	arg = strings.TrimSpace(arg[:closeIdx])
	args := strings.Split(arg, ",")
	for i := range args {
		args[i] = strings.TrimSpace(args[i])
	}
	return name, args, nil
}

func RandStringRunes(n int64, s string) string {
	if len(s) == 0 {
		s = letters
	}
	var letterRunes = []rune(s)
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[randSource.Intn(len(letterRunes))]
	}
	return string(b)
}
