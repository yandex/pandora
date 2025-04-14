package answlog

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Logger struct {
	logger *zap.Logger
	LogSampler
	LogFilter
	loggingEnabled  bool
	samplingEnabled bool
}

func NewLogger(logger *zap.Logger, sampler LogSampler, filter LogFilter, loggingEnabled bool, samplingEnabled bool) *Logger {
	return &Logger{
		logger:          logger,
		LogSampler:      sampler,
		LogFilter:       filter,
		loggingEnabled:  loggingEnabled,
		samplingEnabled: samplingEnabled,
	}
}

func NewNop() *Logger {
	return &Logger{
		logger:     zap.NewNop(),
		LogSampler: &PassAllSampler{},
		LogFilter:  &PassAllFilter{},
	}
}

func (l *Logger) Report(msg string, fields []zapcore.Field, lazyFields func() []zapcore.Field) {
	if !l.loggingEnabled || !l.FilterAnsw(fields) || (l.samplingEnabled && !l.SampleAnsw(fields)) {
		return
	}

	if lazyFields != nil {
		addFields := lazyFields()
		fields = append(fields, addFields...)
	}

	l.logger.Debug(msg, fields...)
}

type LoggerOpt func(*Logger)

func WithSampler(sampler LogSampler, enabled bool) LoggerOpt {
	return func(l *Logger) {
		l.samplingEnabled = enabled
		l.LogSampler = sampler
	}
}

func WithFilter(filter LogFilter) LoggerOpt {
	return func(l *Logger) {
		l.LogFilter = filter
	}
}

func Init(path string, enabled bool, opts ...LoggerOpt) *Logger {
	zapLog := zap.NewNop()
	if enabled {
		writerSyncer := getAnswWriter(path)
		encoder := zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig())

		core := zapcore.NewCore(encoder, writerSyncer, zapcore.DebugLevel)
		zapLog = zap.New(core)
		defer zapLog.Sync()
	}

	logger := NewLogger(zapLog, &PassAllSampler{}, &PassAllFilter{}, enabled, false)
	for _, opt := range opts {
		opt(logger)
	}

	return logger
}

func getAnswWriter(path string) zapcore.WriteSyncer {
	if path == "" {
		path = "./answ.log"
	}
	file, _ := os.Create(path)
	return zapcore.AddSync(file)
}
