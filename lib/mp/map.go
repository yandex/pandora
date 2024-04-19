package mp

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

type ErrSegmentNotFound struct {
	path    string
	segment string
}

func (e *ErrSegmentNotFound) Error() string {
	return fmt.Sprintf("segment %s not found in path %s", e.segment, e.path)
}

func GetMapValue(current map[string]any, path string, iter Iterator) (any, error) {
	if current == nil {
		return nil, nil
	}
	var curSegment strings.Builder
	segments := strings.Split(strings.TrimPrefix(path, "."), ".")

	for i, segment := range segments {
		segment = strings.TrimSpace(segment)
		curSegment.WriteByte('.')
		curSegment.WriteString(segment)
		if strings.Contains(segment, "[") && strings.HasSuffix(segment, "]") {
			openBraceIdx := strings.Index(segment, "[")
			indexStr := strings.ToLower(strings.TrimSpace(segment[openBraceIdx+1 : len(segment)-1]))

			segment = segment[:openBraceIdx]
			pathVal, ok := current[segment]
			if !ok {
				return nil, &ErrSegmentNotFound{path: path, segment: segment}
			}
			sliceElement, err := extractFromSlice(pathVal, indexStr, curSegment.String(), iter)
			if err != nil {
				return nil, fmt.Errorf("cant extract value path=`%s`,segment=`%s`,err=%w", segment, path, err)
			}
			current, ok = sliceElement.(map[string]any)
			if !ok {
				if i != len(segments)-1 {
					return nil, fmt.Errorf("not last segment %s in path %s", segment, path)
				}
				return sliceElement, nil
			}
		} else {
			pathVal, ok := current[segment]
			if !ok {
				return nil, &ErrSegmentNotFound{path: path, segment: segment}
			}
			current, ok = pathVal.(map[string]any)
			if !ok {
				if i != len(segments)-1 {
					return nil, fmt.Errorf("not last segment %s in path %s", segment, path)
				}
				return pathVal, nil
			}
		}
	}

	return current, nil
}

func extractFromSlice(curValue any, indexStr string, curSegment string, iter Iterator) (result any, err error) {
	validTypes := []reflect.Type{
		reflect.TypeOf([]map[string]string{}),
		reflect.TypeOf([]map[string]any{}),
		reflect.TypeOf([]any{}),
		reflect.TypeOf([]string{}),
		reflect.TypeOf([]int{}),
		reflect.TypeOf([]int64{}),
		reflect.TypeOf([]float64{}),
	}

	var valueLen int
	var valueFound bool
	for _, valueType := range validTypes {
		if reflect.TypeOf(curValue) == valueType {
			valueLen = reflect.ValueOf(curValue).Len()
			valueFound = true
			break
		}
	}

	if !valueFound {
		return nil, fmt.Errorf("invalid type of value `%+v`, %T", curValue, curValue)
	}

	index, err := calcIndex(indexStr, curSegment, valueLen, iter)
	if err != nil {
		return nil, fmt.Errorf("failed to calc index for %T; err: %w", curValue, err)
	}

	switch v := curValue.(type) {
	case []map[string]string:
		currentData := make(map[string]any, len(v[index]))
		for k, val := range v[index] {
			currentData[k] = val
		}
		return currentData, nil
	case []map[string]any:
		return v[index], nil
	case []any:
		return v[index], nil
	case []string:
		return v[index], nil
	case []int:
		return v[index], nil
	case []int64:
		return v[index], nil
	case []float64:
		return v[index], nil
	}

	// This line should never be reached, as we've covered all valid types above
	return nil, fmt.Errorf("invalid type of value `%+v`, %T", curValue, curValue)
}

func calcIndex(indexStr string, segment string, length int, iter Iterator) (int, error) {
	index, err := strconv.Atoi(indexStr)
	if err != nil && indexStr != "next" && indexStr != "rand" && indexStr != "last" {
		return 0, fmt.Errorf("index should be integer or one of [next, rand, last], but got `%s`", indexStr)
	}
	if indexStr != "next" && indexStr != "rand" && indexStr != "last" {
		if index >= 0 && index < length {
			return index, nil
		}
		index %= length
		if index < 0 {
			index += length
		}
		return index, nil
	}

	if indexStr == "last" {
		return length - 1, nil
	}
	if indexStr == "rand" {
		return iter.Rand(length), nil
	}
	index = iter.Next(segment)
	if index >= length {
		index %= length
	}
	return index, nil
}
