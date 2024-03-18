package config

import (
	"net"
	"regexp"
	"time"

	"github.com/asaskevich/govalidator"
	"github.com/c2h5oh/datasize"
	validator "gopkg.in/bluesuncorp/validator.v9"
)

func MinTimeValidation(fl validator.FieldLevel) bool {
	t, min, ok := getTimeForValidation(fl.Field().Interface(), fl.Param())
	return ok && min <= t
}
func MaxTimeValidation(fl validator.FieldLevel) bool {
	t, max, ok := getTimeForValidation(fl.Field().Interface(), fl.Param())
	return ok && t <= max
}

func getTimeForValidation(v interface{}, param string) (actual time.Duration, check time.Duration, ok bool) {
	check, err := time.ParseDuration(param)
	if err != nil {
		return
	}
	actual, ok = v.(time.Duration)
	return
}

func MinSizeValidation(fl validator.FieldLevel) bool {
	t, min, ok := getSizeForValidation(fl.Field().Interface(), fl.Param())
	return ok && min <= t
}
func MaxSizeValidation(fl validator.FieldLevel) bool {
	t, max, ok := getSizeForValidation(fl.Field().Interface(), fl.Param())
	return ok && t <= max
}

func getSizeForValidation(v interface{}, param string) (actual, check datasize.ByteSize, ok bool) {
	err := check.UnmarshalText([]byte(param))
	if err != nil {
		return
	}
	actual, ok = v.(datasize.ByteSize)
	return
}

// "host:port" or ":port"
func EndpointStringValidation(value string) bool {
	host, port, err := net.SplitHostPort(value)
	return err == nil &&
		(host == "" || govalidator.IsHost(host)) &&
		govalidator.IsPort(port)
}

// pathRegexp is regexp for at least one path component.
// Valid characters are taken from RFC 3986.
var pathRegexp = regexp.MustCompile(`^(/[a-zA-Z0-9._~!$&'()*+,;=:@%-]+)+$`)

func URLPathStringValidation(value string) bool {
	return pathRegexp.MatchString(value)
}
