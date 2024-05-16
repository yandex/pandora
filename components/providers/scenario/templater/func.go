package templater

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/gofrs/uuid"
	"github.com/yandex/pandora/lib/numbers"
	"github.com/yandex/pandora/lib/str"
)

const defaultMaxRandValue = 10

func init() {
	rand.New(rand.NewSource(time.Now().UnixNano()))
}

type templateFunc func(args ...any) (string, error)

var _ = []templateFunc{
	RandInt,
	RandString,
	UUID,
}

func ParseFunc(v string) (f any, args []string) {
	name, args := parseStr(v)
	if f, ok := GetFuncs()[name]; ok {
		return f, args
	}
	return nil, nil
}

func parseStr(v string) (string, []string) {
	args := strings.Split(v, "(")
	if len(args) == 0 {
		return v, nil
	}
	name := args[0]
	if len(args) == 1 {
		return name, nil
	}
	v = strings.TrimSuffix(strings.Join(args[1:], "("), ")")
	args = strings.Split(v, ",")
	if len(args) == 1 && args[0] == "" {
		return name, nil
	}
	for i := 0; i < len(args); i++ {
		args[i] = strings.TrimSpace(args[i])
	}

	return name, args
}

func GetFuncs() template.FuncMap {
	return map[string]any{
		"randInt":    RandInt,
		"randString": RandString,
		"uuid":       UUID,
	}
}

func RandInt(args ...any) (string, error) {
	switch len(args) {
	case 0:
		return randInt(0, 0)
	case 1:
		f, err := numbers.ParseInt(args[0])
		if err != nil {
			return "", err
		}
		return randInt(f, 0)
	case 2:
		f, err := numbers.ParseInt(args[0])
		if err != nil {
			return "", err
		}
		t, err := numbers.ParseInt(args[1])
		if err != nil {
			return "", err
		}
		return randInt(f, t)
	default:
		return "", fmt.Errorf("maximum 2 arguments expected but got %d", len(args))
	}
}

func randInt(f, t int64) (string, error) {
	if t < f {
		t, f = f, t
	}
	if f == 0 && t == 0 {
		t = defaultMaxRandValue
	}
	if t == f {
		f = t + defaultMaxRandValue
	}
	n := rand.Int63n(t - f)
	n += f
	return strconv.FormatInt(n, 10), nil
}

func RandString(args ...any) (string, error) {
	switch len(args) {
	case 0:
		return randString(0, "")
	case 1:
		return randString(args[0], "")
	case 2:
		return randString(args[0], str.FormatString(args[1]))
	default:
		return "", fmt.Errorf("maximum 2 arguments expected but got %d", len(args))
	}
}

func randString(cnt any, letters string) (string, error) {
	n, err := numbers.ParseInt(cnt)
	if err != nil {
		return "", err
	}
	if n == 0 {
		n = 1
	}
	return str.RandStringRunes(n, letters), nil
}

func UUID(args ...any) (string, error) {
	v, err := uuid.NewV4()
	if err != nil {
		return "", err
	}
	return v.String(), nil
}
