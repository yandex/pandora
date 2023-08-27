package str

import (
	"errors"
	"strings"
)

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
