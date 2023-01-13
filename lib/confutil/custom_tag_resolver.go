package confutil

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

var (
	notoken                         = ""
	ErrNoTagsFound                  = errors.New("no tags found")
	ErrUnsupportedKind              = errors.New("unsupported kind")
	ErrCantCastVariableToTargetType = errors.New("can't cast variable")
	ErrResolverNotRegistered        = errors.New("unknown tag type")
)

type TagResolver func(string) (string, error)

type tagEntry struct {
	tagType string
	string  string
	varname string
}

var resolvers map[string]TagResolver = make(map[string]TagResolver)

// Register custom tag resolver for config variables
func RegisterTagResolver(tagType string, resolver TagResolver) {
	tagType = strings.ToLower(tagType)
	// silent overwrite existing resolver
	resolvers[tagType] = resolver
}

func getTagResolver(tagType string) (TagResolver, error) {
	tagType = strings.ToLower(tagType)
	r, ok := resolvers[tagType]
	if !ok {
		return nil, ErrResolverNotRegistered
	}
	return r, nil
}

// Resolve config variables in format ${tagType:variable}
func ResolveCustomTags(s string, targetType reflect.Type) (interface{}, error) {
	tokens, err := findTags(s)
	if err != nil {
		return nil, err
	}

	if len(tokens) == 0 {
		return s, ErrNoTagsFound
	}

	res := s
	for _, t := range tokens {
		resolver, err := getTagResolver(t.tagType)
		if err == ErrResolverNotRegistered {
			continue
		} else if err != nil {
			return nil, err
		}

		resolved, err := resolver(t.varname)
		if err != nil {
			return nil, err
		}
		res = strings.ReplaceAll(res, t.string, resolved)
	}

	// try to cast result to target type, because mapstructure will not attempt to cast strings to bool, int and floats
	// if target type is unknown, we still let other hooks process result (time.Duration, ipv4 and other hooks will do)
	if len(tokens) == 1 && strings.TrimSpace(s) == tokens[0].string {
		castedRes, err := cast(res, targetType)
		if err == nil || !errors.Is(err, ErrCantCastVariableToTargetType) {
			return castedRes, err
		}
	}
	return res, nil
}

func findTags(s string) ([]*tagEntry, error) {
	tagRegexp := regexp.MustCompile(`\$\{(?:([^}]+?):)?([^{}]+?)\}`)
	tokensFound := tagRegexp.FindAllStringSubmatch(s, -1)
	result := make([]*tagEntry, 0, len(tokensFound))

	for _, token := range tokensFound {
		tag := &tagEntry{
			tagType: strings.TrimSpace(token[1]),
			varname: strings.TrimSpace(token[2]),
			string:  token[0],
		}
		result = append(result, tag)
	}

	return result, nil
}

func cast(v string, t reflect.Type) (interface{}, error) {
	switch t.Kind() {
	case reflect.Bool:
		return castBool(v)
	case reflect.Int,
		reflect.Int8,
		reflect.Int16,
		reflect.Int32,
		reflect.Int64,
		reflect.Uint,
		reflect.Uint8,
		reflect.Uint16,
		reflect.Uint32,
		reflect.Uint64:
		return castInt(v, t)
	case reflect.Float32,
		reflect.Float64:
		return castFloat(v, t)
	case reflect.String:
		return v, nil
	}
	return nil, ErrUnsupportedKind
}

func castBool(v string) (interface{}, error) {
	res, err := strconv.ParseBool(v)
	if err != nil {
		return false, fmt.Errorf("'%s' cast to bool failed: %w", v, ErrCantCastVariableToTargetType)
	}

	return res, nil
}

func castInt(v string, t reflect.Type) (interface{}, error) {
	intV, err := strconv.ParseInt(v, 0, t.Bits())
	if err != nil {
		return nil, fmt.Errorf("'%s' cast to %s failed: %w", v, t, ErrCantCastVariableToTargetType)
	}

	switch t.Kind() {
	case reflect.Int:
		return int(intV), nil
	case reflect.Int8:
		return int8(intV), nil
	case reflect.Int16:
		return int16(intV), nil
	case reflect.Int32:
		return int32(intV), nil
	case reflect.Int64:
		return int64(intV), nil
	case reflect.Uint:
		return uint(intV), nil
	case reflect.Uint8:
		return uint8(intV), nil
	case reflect.Uint16:
		return uint16(intV), nil
	case reflect.Uint32:
		return uint32(intV), nil
	case reflect.Uint64:
		return uint64(intV), nil
	}

	return nil, ErrUnsupportedKind
}

func castFloat(v string, t reflect.Type) (interface{}, error) {
	intV, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return 0.0, fmt.Errorf("'%s' cast to %s failed: %w", v, t, ErrCantCastVariableToTargetType)
	}

	switch t.Kind() {
	case reflect.Float32:
		return float32(intV), nil
	case reflect.Float64:
		return float64(intV), nil
	}

	return nil, ErrUnsupportedKind
}
