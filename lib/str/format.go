package str

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
)

func FormatString(iface any) string {
	switch val := iface.(type) {
	case []byte:
		return string(val)
	}
	v := reflect.ValueOf(iface)
	switch v.Kind() {
	case reflect.Invalid:
		return ""
	case reflect.Bool:
		return strconv.FormatBool(v.Bool())
	case reflect.String:
		return v.String()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(v.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return strconv.FormatUint(v.Uint(), 10)
	case reflect.Float64:
		return strconv.FormatFloat(v.Float(), 'f', -1, 64)
	case reflect.Float32:
		return strconv.FormatFloat(v.Float(), 'f', -1, 32)
	case reflect.Ptr:
		b, err := json.Marshal(v.Interface())
		if err != nil {
			return "nil"
		}
		return string(b)
	}
	return fmt.Sprintf("%v", iface)
}
