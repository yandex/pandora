package answlog

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func Init(path string) *zap.Logger {
	writerSyncer := getAnswWriter(path)
	encoder := zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig())
	core := zapcore.NewCore(encoder, writerSyncer, zapcore.DebugLevel)

	Log := zap.New(core)
	defer Log.Sync()
	return Log
}

func getAnswWriter(path string) zapcore.WriteSyncer {
	if path == "" {
		path = "./answ.log"
	}
	file, _ := os.Create(path)
	return zapcore.AddSync(file)
}
