package answlog

import (
	"fmt"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Logger struct {
	logger *zap.Logger
	LogSampler
	LogFilter
	LogMasker
	loggingEnabled  bool
	samplingEnabled bool
}

func NewLogger(logger *zap.Logger, sampler LogSampler, filter LogFilter, masker LogMasker, loggingEnabled bool, samplingEnabled bool) *Logger {
	return &Logger{
		logger:          logger,
		LogSampler:      sampler,
		LogFilter:       filter,
		LogMasker:       masker,
		loggingEnabled:  loggingEnabled,
		samplingEnabled: samplingEnabled,
	}
}

func NewNop() *Logger {
	return &Logger{
		logger:     zap.NewNop(),
		LogSampler: &PassAllSampler{},
		LogFilter:  &PassAllFilter{},
		LogMasker:  &PassAllMasker{},
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

	fields, err := l.LogMasker.Mask(fields)
	if err != nil {
		l.logger.Debug(fmt.Sprintf("Some error occurred while trying to mask fields: %s. Skipping log entry", err))
		return
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

func WithMasker(masker LogMasker) LoggerOpt {
	return func(l *Logger) {
		l.LogMasker = masker
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

	logger := NewLogger(zapLog, &PassAllSampler{}, &PassAllFilter{}, &PassAllMasker{}, enabled, false)
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
