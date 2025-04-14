package answ

import "go.uber.org/zap/zapcore"

func GetField(fields []zapcore.Field, fieldKey string) (zapcore.Field, bool) {
	for _, f := range fields {
		if f.Key == fieldKey {
			return f, true
		}
	}
	return zapcore.Field{}, false
}
