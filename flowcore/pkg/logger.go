package pkg

import (
	"os"
	"strings"

	"go.temporal.io/sdk/log"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// ZapLogger 使用 zap 实现 Temporal 的日志接口
type ZapLogger struct {
	logger *zap.Logger
	level  LogLevel
}

type LogLevel int

const (
	LevelDebug LogLevel = iota
	LevelInfo
	LevelWarn
	LevelError
)

// NewLogger 创建一个新的带有指定日志级别的 zap logger
func NewLogger(level string) log.Logger {
	logLevel := parseLogLevel(level)
	zapLogger := createZapLogger(logLevel)

	return &ZapLogger{
		logger: zapLogger,
		level:  logLevel,
	}
}

// NewLoggerWithZap 使用现有的 zap logger 创建一个 logger
func NewLoggerWithZap(zapLogger *zap.Logger, level string) log.Logger {
	logLevel := parseLogLevel(level)
	return &ZapLogger{
		logger: zapLogger,
		level:  logLevel,
	}
}

// Debug 记录调试信息
func (l *ZapLogger) Debug(msg string, keyvals ...interface{}) {
	if l.level <= LevelDebug {
		fields := l.parseKeyvals(keyvals...)
		l.logger.Debug(msg, fields...)
	}
}

// Info 记录信息消息
func (l *ZapLogger) Info(msg string, keyvals ...interface{}) {
	if l.level <= LevelInfo {
		fields := l.parseKeyvals(keyvals...)
		l.logger.Info(msg, fields...)
	}
}

// Warn 记录警告消息
func (l *ZapLogger) Warn(msg string, keyvals ...interface{}) {
	if l.level <= LevelWarn {
		fields := l.parseKeyvals(keyvals...)
		l.logger.Warn(msg, fields...)
	}
}

// Error 记录错误消息
func (l *ZapLogger) Error(msg string, keyvals ...interface{}) {
	if l.level <= LevelError {
		fields := l.parseKeyvals(keyvals...)
		l.logger.Error(msg, fields...)
	}
}

// parseKeyvals 将 keyvals 转换为 zap 字段
func (l *ZapLogger) parseKeyvals(keyvals ...interface{}) []zap.Field {
	if len(keyvals) == 0 {
		return nil
	}

	fields := make([]zap.Field, 0, len(keyvals)/2)

	for i := 0; i < len(keyvals); i += 2 {
		if i+1 < len(keyvals) {
			key := toString(keyvals[i])
			value := keyvals[i+1]
			fields = append(fields, zap.Any(key, value))
		} else {
			// Handle odd number of keyvals
			key := toString(keyvals[i])
			fields = append(fields, zap.String(key, "<missing>"))
		}
	}

	return fields
}

// toString 安全地将 interface{} 转换为 string
func toString(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return "unknown"
}

// createZapLogger 创建配置好的 zap logger
func createZapLogger(level LogLevel) *zap.Logger {
	// Configure encoder
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "timestamp"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	encoderConfig.MessageKey = "message"
	encoderConfig.CallerKey = "caller"
	encoderConfig.EncodeCaller = zapcore.ShortCallerEncoder

	// Configure core
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.AddSync(zapcore.Lock(zapcore.AddSync(zapcore.NewMultiWriteSyncer(zapcore.AddSync(os.Stdout))))),
		mapToZapLevel(level),
	)

	// Create logger with caller info
	logger := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))

	// Add temporal prefix to all logs
	logger = logger.Named("temporal")

	return logger
}

// createDevelopmentZapLogger 创建适合开发环境的 zap logger
func createDevelopmentZapLogger(level LogLevel) *zap.Logger {
	// Configure encoder for development (console output)
	encoderConfig := zap.NewDevelopmentEncoderConfig()
	encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeCaller = zapcore.ShortCallerEncoder

	// Configure core
	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(encoderConfig),
		zapcore.AddSync(os.Stdout),
		mapToZapLevel(level),
	)

	// Create logger
	logger := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
	logger = logger.Named("temporal")

	return logger
}

// NewDevelopmentLogger 创建适合开发环境的日志记录器
func NewDevelopmentLogger(level string) log.Logger {
	logLevel := parseLogLevel(level)
	zapLogger := createDevelopmentZapLogger(logLevel)

	return &ZapLogger{
		logger: zapLogger,
		level:  logLevel,
	}
}

// mapToZapLevel 将我们的 LogLevel 转换为 zap Level
func mapToZapLevel(level LogLevel) zapcore.Level {
	switch level {
	case LevelDebug:
		return zapcore.DebugLevel
	case LevelInfo:
		return zapcore.InfoLevel
	case LevelWarn:
		return zapcore.WarnLevel
	case LevelError:
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}

// parseLogLevel 解析字符串级别的日志级别
func parseLogLevel(level string) LogLevel {
	switch strings.ToLower(level) {
	case "debug":
		return LevelDebug
	case "info":
		return LevelInfo
	case "warn", "warning":
		return LevelWarn
	case "error":
		return LevelError
	default:
		return LevelInfo
	}
}

// WithFields 创建一个带有附加字段的新日志记录器
func (l *ZapLogger) WithFields(fields map[string]interface{}) log.Logger {
	zapFields := make([]zap.Field, 0, len(fields))
	for k, v := range fields {
		zapFields = append(zapFields, zap.Any(k, v))
	}

	newLogger := l.logger.With(zapFields...)
	return &ZapLogger{
		logger: newLogger,
		level:  l.level,
	}
}

// Sync 刷新任何缓冲的日志条目
func (l *ZapLogger) Sync() error {
	return l.logger.Sync()
}

// GetZapLogger 返回底层的 zap logger，用于高级用法
func (l *ZapLogger) GetZapLogger() *zap.Logger {
	return l.logger
}

// SetLevel 更改日志级别
func (l *ZapLogger) SetLevel(level string) {
	l.level = parseLogLevel(level)
}

// IsDebugEnabled 如果启用了调试级别，则返回 true
func (l *ZapLogger) IsDebugEnabled() bool {
	return l.level <= LevelDebug
}

// IsInfoEnabled 如果启用了信息级别，则返回 true
func (l *ZapLogger) IsInfoEnabled() bool {
	return l.level <= LevelInfo
}

// Factory functions for common logger configurations

// NewProductionLogger 创建生产就绪的日志记录器
func NewProductionLogger() log.Logger {
	config := zap.NewProductionConfig()
	config.OutputPaths = []string{"stdout"}
	config.ErrorOutputPaths = []string{"stderr"}

	logger, _ := config.Build(zap.AddCallerSkip(1))
	logger = logger.Named("temporal")

	return &ZapLogger{
		logger: logger,
		level:  LevelInfo,
	}
}

// NewFileLogger 创建一个写入文件的日志记录器
func NewFileLogger(filepath string, level string) (log.Logger, error) {
	config := zap.NewProductionConfig()
	config.OutputPaths = []string{filepath}
	config.ErrorOutputPaths = []string{filepath}
	config.Level = zap.NewAtomicLevelAt(mapToZapLevel(parseLogLevel(level)))

	logger, err := config.Build(zap.AddCallerSkip(1))
	if err != nil {
		return nil, err
	}

	logger = logger.Named("temporal")

	return &ZapLogger{
		logger: logger,
		level:  parseLogLevel(level),
	}, nil
}

// NewStructuredLogger 创建一个具有预定义结构字段的日志记录器
func NewStructuredLogger(level string, fields map[string]interface{}) log.Logger {
	baseLogger := NewLogger(level).(*ZapLogger)
	return baseLogger.WithFields(fields)
}
