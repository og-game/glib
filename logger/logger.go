package logger

import (
	"context"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

// Logger 日志器，完全兼容 logx.Logger
type Logger struct {
	logx.Logger
}

// New 创建新的日志器
func New(ctx context.Context) *Logger {
	return &Logger{
		Logger: logx.WithContext(ctx),
	}
}

// WithContext 创建带上下文的日志器
func WithContext(ctx context.Context) *Logger {
	return New(ctx)
}

// ========== 标记方法（可选使用）==========
// 这些方法只是在日志中添加一个 mark 字段，方便识别
// 使用与否完全取决于业务需要

// MarkCritical 标记为关键日志
func (l *Logger) MarkCritical() *Logger {
	return &Logger{
		Logger: l.Logger.WithFields(logx.Field("mark", "critical")),
	}
}

// MarkBusiness 标记为业务日志
func (l *Logger) MarkBusiness() *Logger {
	return &Logger{
		Logger: l.Logger.WithFields(logx.Field("mark", "business")),
	}
}

// ========== 原生方法（保持不变）==========

// Debug logs a message at debug level
func (l *Logger) Debug(v ...any) {
	l.Logger.Debug(v...)
}

// Debugf logs a message at debug level
func (l *Logger) Debugf(format string, v ...any) {
	l.Logger.Debugf(format, v...)
}

// Debugv logs a message at debug level
func (l *Logger) Debugv(v any) {
	l.Logger.Debugv(v)
}

// Debugw logs a message at debug level
func (l *Logger) Debugw(msg string, fields ...logx.LogField) {
	l.Logger.Debugw(msg, fields...)
}

// Info logs a message at info level
func (l *Logger) Info(v ...any) {
	l.Logger.Info(v...)
}

// Infof logs a message at info level
func (l *Logger) Infof(format string, v ...any) {
	l.Logger.Infof(format, v...)
}

// Infov logs a message at info level
func (l *Logger) Infov(v any) {
	l.Logger.Infov(v)
}

// Infow logs a message at info level
func (l *Logger) Infow(msg string, fields ...logx.LogField) {
	l.Logger.Infow(msg, fields...)
}

// Error logs a message at error level
func (l *Logger) Error(v ...any) {
	l.Logger.Error(v...)
}

// Errorf logs a message at error level
func (l *Logger) Errorf(format string, v ...any) {
	l.Logger.Errorf(format, v...)
}

// Errorv logs a message at error level
func (l *Logger) Errorv(v any) {
	l.Logger.Errorv(v)
}

// Errorw logs a message at error level
func (l *Logger) Errorw(msg string, fields ...logx.LogField) {
	l.Logger.Errorw(msg, fields...)
}

// Slow logs a message at slow level
func (l *Logger) Slow(v ...any) {
	l.Logger.Slow(v...)
}

// Slowf logs a message at slow level
func (l *Logger) Slowf(format string, v ...any) {
	l.Logger.Slowf(format, v...)
}

// Slowv logs a message at slow level
func (l *Logger) Slowv(v any) {
	l.Logger.Slowv(v)
}

// Sloww logs a message at slow level
func (l *Logger) Sloww(msg string, fields ...logx.LogField) {
	l.Logger.Sloww(msg, fields...)
}

// WithCallerSkip returns a new logger with the given caller skip
func (l *Logger) WithCallerSkip(skip int) *Logger {
	return &Logger{
		Logger: l.Logger.WithCallerSkip(skip),
	}
}

// WithContext returns a new logger with the given context
func (l *Logger) WithContext(ctx context.Context) *Logger {
	return &Logger{
		Logger: l.Logger.WithContext(ctx),
	}
}

// WithDuration returns a new logger with the given duration
func (l *Logger) WithDuration(d time.Duration) *Logger {
	return &Logger{
		Logger: l.Logger.WithDuration(d),
	}
}

// WithFields returns a new logger with the given fields
func (l *Logger) WithFields(fields ...logx.LogField) *Logger {
	return &Logger{
		Logger: l.Logger.WithFields(fields...),
	}
}
