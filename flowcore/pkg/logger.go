package pkg

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

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

// ANSI 颜色代码
const (
	ColorReset   = "\033[0m"
	ColorRed     = "\033[31m"
	ColorGreen   = "\033[32m"
	ColorYellow  = "\033[33m"
	ColorBlue    = "\033[34m"
	ColorMagenta = "\033[35m"
	ColorCyan    = "\033[36m"
	ColorBold    = "\033[1m"

	// 粗体颜色组合
	ColorRedBold     = "\033[1;31m"
	ColorGreenBold   = "\033[1;32m"
	ColorYellowBold  = "\033[1;33m"
	ColorBlueBold    = "\033[1;34m"
	ColorMagentaBold = "\033[1;35m"
	ColorCyanBold    = "\033[1;36m"
)

var (
	projectRootOnce sync.Once
	projectRootPath string
	enableColor     = isColorTerminal()
)

// LoggerConfig 日志配置
type LoggerConfig struct {
	Level       string // debug, info, warn, error
	WithIcon    bool   // 是否显示图标
	WithColor   bool   // 是否启用颜色
	Format      string // console, json
	ShowCaller  bool   // 是否显示调用者信息
	ProjectRoot string // 项目根目录
}

// 默认配置
func defaultConfig() LoggerConfig {
	return LoggerConfig{
		Level:      "info",
		WithIcon:   true,
		WithColor:  true,
		Format:     "console",
		ShowCaller: true,
	}
}

// isColorTerminal 检测终端是否支持颜色
func isColorTerminal() bool {
	term := os.Getenv("TERM")
	colorTerm := os.Getenv("COLORTERM")

	if colorTerm != "" {
		return true
	}

	colorTerms := []string{"xterm", "screen", "tmux", "rxvt", "ansi"}
	for _, ct := range colorTerms {
		if strings.Contains(term, ct) {
			return true
		}
	}
	return false
}

// getColor 根据配置返回颜色代码
func getColor(color string) string {
	if enableColor {
		return color
	}
	return ""
}

// customTimeEncoder 时间编码器
func customTimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Format("2006-01-02 15:04:05.000"))
}

// customLevelEncoder 级别编码器
func customLevelEncoder(withIcon bool) zapcore.LevelEncoder {
	return func(level zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
		var levelStr string
		switch level {
		case zapcore.DebugLevel:
			if withIcon {
				levelStr = getColor(ColorCyanBold) + "🔍 [DEBUG]" + getColor(ColorReset)
			} else {
				levelStr = "[DEBUG]"
			}
		case zapcore.InfoLevel:
			if withIcon {
				levelStr = getColor(ColorGreenBold) + "ℹ️ [INFO]" + getColor(ColorReset)
			} else {
				levelStr = "[INFO] "
			}
		case zapcore.WarnLevel:
			if withIcon {
				levelStr = getColor(ColorYellowBold) + "⚠️ [WARN]" + getColor(ColorReset)
			} else {
				levelStr = "[WARN] "
			}
		case zapcore.ErrorLevel:
			if withIcon {
				levelStr = getColor(ColorRedBold) + "🚨 [ERROR]" + getColor(ColorReset)
			} else {
				levelStr = "[ERROR]"
			}
		default:
			levelStr = "[UNKNOWN]"
		}
		enc.AppendString(levelStr)
	}
}

// customCallerEncoder 调用者编码器
func customCallerEncoder(withIcon bool) zapcore.CallerEncoder {
	return func(caller zapcore.EntryCaller, enc zapcore.PrimitiveArrayEncoder) {
		file := getRelativePath(caller.File)

		pc := caller.PC
		fn := runtime.FuncForPC(pc)
		funcName := "unknown"
		if fn != nil {
			funcName = fn.Name()
			if idx := strings.LastIndex(funcName, "."); idx != -1 {
				funcName = funcName[idx+1:]
			}
		}

		if withIcon {
			// 📂 文件路径 - 蓝色粗体，函数名 - 紫色粗体
			fileStr := getColor(ColorBlueBold) + fmt.Sprintf("📂 %s:%d", file, caller.Line) + getColor(ColorReset)
			funcStr := getColor(ColorMagentaBold) + funcName + getColor(ColorReset)
			enc.AppendString(fmt.Sprintf("%s:%s", fileStr, funcStr))
		} else {
			enc.AppendString(fmt.Sprintf("@ %s:%d:%s", file, caller.Line, funcName))
		}
	}
}

// findProjectRoot 查找项目根目录
func findProjectRoot() string {
	projectRootOnce.Do(func() {
		_, currentFile, _, ok := runtime.Caller(0)
		if !ok {
			if wd, err := os.Getwd(); err == nil {
				projectRootPath = wd
			} else {
				projectRootPath = "."
			}
			return
		}

		dir := filepath.Dir(currentFile)
		for {
			for _, marker := range []string{"go.mod", ".git", "Makefile", "README.md"} {
				if _, err := os.Stat(filepath.Join(dir, marker)); err == nil {
					projectRootPath = dir
					return
				}
			}

			parent := filepath.Dir(dir)
			if parent == dir {
				if wd, err := os.Getwd(); err == nil {
					projectRootPath = wd
				} else {
					projectRootPath = "."
				}
				return
			}
			dir = parent
		}
	})
	return projectRootPath
}

// getRelativePath 获取相对路径
func getRelativePath(fullPath string) string {
	projectRoot := findProjectRoot()

	if relPath, err := filepath.Rel(projectRoot, fullPath); err == nil {
		if !strings.HasPrefix(relPath, "..") {
			cleanPath := filepath.ToSlash(relPath)

			// 智能压缩长路径
			parts := strings.Split(cleanPath, "/")
			if len(parts) > 4 {
				cleanPath = parts[0] + "/.../" + strings.Join(parts[len(parts)-2:], "/")
			}
			return cleanPath
		}
	}

	// 兜底方案
	parts := strings.Split(filepath.ToSlash(fullPath), "/")
	if len(parts) >= 3 {
		return strings.Join(parts[len(parts)-3:], "/")
	}
	return filepath.Base(fullPath)
}

// isFrameworkCode 判断是否是框架代码
func isFrameworkCode(file string) bool {
	frameworkPaths := []string{
		"/go/pkg/mod/", "/usr/local/go/src/", "go.temporal.io",
		"go.uber.org/zap", "/runtime/", "/reflect/", "src/",
	}

	for _, framework := range frameworkPaths {
		if strings.Contains(file, framework) {
			return true
		}
	}

	return strings.HasSuffix(file, "logger.go")
}

// findUserCode 查找用户代码位置
func findUserCode(skip int) (string, int, string, bool) {
	for i := skip; i < skip+10; i++ {
		pc, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}

		if isFrameworkCode(file) {
			continue
		}

		fn := runtime.FuncForPC(pc)
		funcName := "unknown"
		if fn != nil {
			funcName = fn.Name()
			if idx := strings.LastIndex(funcName, "."); idx != -1 {
				funcName = funcName[idx+1:]
			}
		}

		return file, line, funcName, true
	}
	return "unknown", 0, "unknown", false
}

// NewLogger 创建默认日志记录器
func NewLogger(level string) log.Logger {
	config := defaultConfig()
	config.Level = level
	return NewLoggerWithConfig(config)
}

// NewLoggerWithConfig 使用配置创建 logger
func NewLoggerWithConfig(config LoggerConfig) log.Logger {
	logLevel := parseLogLevel(config.Level)

	if config.ProjectRoot != "" {
		projectRootPath = config.ProjectRoot
	}

	// 设置颜色
	if config.WithColor {
		enableColor = true
	}

	zapLogger := createZapLogger(logLevel, config)

	return &ZapLogger{
		logger: zapLogger,
		level:  logLevel,
	}
}

// createZapLogger 创建 zap logger
func createZapLogger(level LogLevel, config LoggerConfig) *zap.Logger {
	var encoder zapcore.Encoder

	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeTime:     customTimeEncoder,
		EncodeLevel:    customLevelEncoder(config.WithIcon),
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeName:     zapcore.FullNameEncoder,
	}

	if config.ShowCaller {
		encoderConfig.EncodeCaller = customCallerEncoder(config.WithIcon)
	}

	if config.Format == "json" {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	} else {
		encoderConfig.ConsoleSeparator = " | "
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}

	core := zapcore.NewCore(
		encoder,
		zapcore.AddSync(os.Stdout),
		mapToZapLevel(level),
	)

	logger := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(3))
	return logger.Named("temporal")
}

// Debug 记录调试信息
func (l *ZapLogger) Debug(msg string, keyvals ...interface{}) {
	if l.level <= LevelDebug {
		fields := l.parseKeyvals(keyvals...)
		fields = append(fields, l.getCallerField())
		l.logger.Debug(msg, fields...)
	}
}

// Info 记录信息消息
func (l *ZapLogger) Info(msg string, keyvals ...interface{}) {
	if l.level <= LevelInfo {
		fields := l.parseKeyvals(keyvals...)
		fields = append(fields, l.getCallerField())
		l.logger.Info(msg, fields...)
	}
}

// Warn 记录警告消息
func (l *ZapLogger) Warn(msg string, keyvals ...interface{}) {
	if l.level <= LevelWarn {
		fields := l.parseKeyvals(keyvals...)
		fields = append(fields, l.getCallerField())
		l.logger.Warn(msg, fields...)
	}
}

// Error 记录错误消息
func (l *ZapLogger) Error(msg string, keyvals ...interface{}) {
	if l.level <= LevelError {
		fields := l.parseKeyvals(keyvals...)
		fields = append(fields, l.getCallerField())
		l.logger.Error(msg, fields...)
	}
}

// getCallerField 获取调用者信息
func (l *ZapLogger) getCallerField() zap.Field {
	file, line, funcName, ok := findUserCode(3)
	if !ok {
		return zap.String("source", "unknown")
	}

	relativePath := getRelativePath(file)

	// 应用颜色：文件路径蓝色粗体，函数名紫色粗体
	if enableColor {
		fileStr := getColor(ColorBlueBold) + fmt.Sprintf("%s:%d", relativePath, line) + getColor(ColorReset)
		funcStr := getColor(ColorMagentaBold) + funcName + getColor(ColorReset)
		return zap.String("source", fmt.Sprintf("%s:%s", fileStr, funcStr))
	}

	return zap.String("source", fmt.Sprintf("%s:%d:%s", relativePath, line, funcName))
}

// parseKeyvals 转换 keyvals 为 zap 字段
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
			key := toString(keyvals[i])
			fields = append(fields, zap.String(key, "<missing>"))
		}
	}

	return fields
}

// toString 安全转换为字符串
func toString(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return "unknown"
}

// mapToZapLevel 转换日志级别
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

// parseLogLevel 解析日志级别
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

// WithFields 创建带附加字段的新日志记录器
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

// 便捷方法

// NewDevelopmentLogger 开发环境日志记录器
func NewDevelopmentLogger(level string) log.Logger {
	return NewLoggerWithConfig(LoggerConfig{
		Level:      level,
		WithIcon:   true,
		WithColor:  true,
		Format:     "console",
		ShowCaller: true,
	})
}

// NewProductionLogger 生产环境日志记录器
func NewProductionLogger() log.Logger {
	return NewLoggerWithConfig(LoggerConfig{
		Level:      "info",
		WithIcon:   false,
		WithColor:  false,
		Format:     "json",
		ShowCaller: true,
	})
}

// NewFileLogger 文件日志记录器
func NewFileLogger(filepath string, level string) (log.Logger, error) {
	config := zap.NewProductionConfig()
	config.OutputPaths = []string{filepath}
	config.Level = zap.NewAtomicLevelAt(mapToZapLevel(parseLogLevel(level)))

	config.EncoderConfig.EncodeTime = customTimeEncoder
	config.EncoderConfig.CallerKey = "caller"
	config.EncoderConfig.EncodeCaller = customCallerEncoder(false)

	logger, err := config.Build(zap.AddCallerSkip(3))
	if err != nil {
		return nil, err
	}

	return &ZapLogger{
		logger: logger.Named("temporal"),
		level:  parseLogLevel(level),
	}, nil
}
