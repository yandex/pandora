package numbers

import (
	"fmt"
	"math"
	"strconv"
)

func ParseInt(input any) (int64, error) {
	switch v := input.(type) {
	case int:
		return int64(v), nil
	case int8:
		return int64(v), nil
	case int16:
		return int64(v), nil
	case int32:
		return int64(v), nil
	case int64:
		return v, nil
	case uint:
		if v > uint(math.MaxInt64) {
			return 0, fmt.Errorf("uint value overflows int64")
		}
		return int64(v), nil
	case uint8:
		return int64(v), nil
	case uint16:
		return int64(v), nil
	case uint32:
		return int64(v), nil
	case uint64:
		if v > uint64(math.MaxInt64) {
			return 0, fmt.Errorf("uint64 value overflows int64")
		}
		return int64(v), nil
	case string:
		f, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("cannot parse '%v' as int64: %w", v, err)
		}
		return f, nil
	default:
		return 0, fmt.Errorf("unsupported type: %T", input)
	}
}

func ParseFloat(input any) (float64, error) {
	switch v := input.(type) {
	case int:
		return float64(v), nil
	case int32:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case uint:
		return float64(v), nil
	case uint32:
		return float64(v), nil
	case uint64:
		if v > uint64(^uint(0)>>1) {
			return 0, fmt.Errorf("uint64 value too large for precise conversion: %v", v)
		}
		return float64(v), nil
	case float32:
		return float64(v), nil
	case float64:
		return v, nil
	case string:
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return 0, fmt.Errorf("cannot parse '%v' as float64: %w", v, err)
		}
		return f, nil
	default:
		return 0, fmt.Errorf("unsupported type: %T", v)
	}
}
