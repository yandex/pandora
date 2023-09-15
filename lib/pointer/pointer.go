package pointer

func ToString(s string) *string {
	return &s
}

func ToBool(b bool) *bool {
	return &b
}

func ToInt64(b int64) *int64 {
	return &b
}
