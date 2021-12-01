package answlog

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func Init() *zap.Logger {
	writerSyncer := getAnswWriter()
	encoder := zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig())
	core := zapcore.NewCore(encoder, writerSyncer, zapcore.DebugLevel)

	Log := zap.New(core)
	defer Log.Sync()
	return Log
}

func getAnswWriter() zapcore.WriteSyncer {
	file, _ := os.Create("./answ.log")
	return zapcore.AddSync(file)
}
