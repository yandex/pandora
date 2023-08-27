package mp

import (
	"fmt"
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

func GetMapValue(input map[string]any, path string, iter Iterator) (any, error) {
	var curSegment strings.Builder
	segments := strings.Split(path, ".")

	currentData := input
	for i, segment := range segments {
		segment = strings.TrimSpace(segment)
		curSegment.WriteByte('.')
		curSegment.WriteString(segment)
		if strings.Contains(segment, "[") && strings.HasSuffix(segment, "]") {
			openBraceIdx := strings.Index(segment, "[")
			indexStr := strings.ToLower(strings.TrimSpace(segment[openBraceIdx+1 : len(segment)-1]))

			segment = segment[:openBraceIdx]
			value, exists := currentData[segment]
			if !exists {
				return nil, &ErrSegmentNotFound{path: path, segment: segment}
			}

			mval, isMval := value.([]map[string]string)
			if isMval {
				index, err := calcIndex(indexStr, curSegment.String(), len(mval), iter)
				if err != nil {
					return nil, fmt.Errorf("failed to calc index: %w", err)
				}
				vval := mval[index]
				currentData = make(map[string]any, len(vval))
				for k, v := range vval {
					currentData[k] = v
				}
				continue
			}

			mapSlice, isMapSlice := value.([]map[string]any)
			if !isMapSlice {
				anySlice, isAnySlice := value.([]any)
				if isAnySlice {
					index, err := calcIndex(indexStr, curSegment.String(), len(anySlice), iter)
					if err != nil {
						return nil, fmt.Errorf("failed to calc index: %w", err)
					}
					if i != len(segments)-1 {
						return nil, fmt.Errorf("not last segment %s in path %s", segment, path)
					}
					return anySlice[index], nil
				}
				stringSlice, isStringSlice := value.([]string)
				if isStringSlice {
					index, err := calcIndex(indexStr, curSegment.String(), len(stringSlice), iter)
					if err != nil {
						return nil, fmt.Errorf("failed to calc index: %w", err)
					}
					if i != len(segments)-1 {
						return nil, fmt.Errorf("not last segment %s in path %s", segment, path)
					}
					return stringSlice[index], nil
				}
				return nil, fmt.Errorf("invalid type of segment %s in path %s", segment, path)
			}

			index, err := calcIndex(indexStr, curSegment.String(), len(mapSlice), iter)
			if err != nil {
				return nil, fmt.Errorf("failed to calc index: %w", err)
			}
			currentData = mapSlice[index]
		} else {
			value, exists := currentData[segment]
			if !exists {
				return nil, &ErrSegmentNotFound{path: path, segment: segment}
			}
			var ok bool
			currentData, ok = value.(map[string]any)
			if !ok {
				if i != len(segments)-1 {
					return nil, fmt.Errorf("not last segment %s in path %s", segment, path)
				}
				return value, nil
			}
		}
	}

	return currentData, nil
}

func calcIndex(indexStr string, segment string, length int, iter Iterator) (int, error) {
	index, err := strconv.Atoi(indexStr)
	if err != nil && indexStr != "next" && indexStr != "rand" && indexStr != "last" {
		return 0, fmt.Errorf("invalid index: %s", indexStr)
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
