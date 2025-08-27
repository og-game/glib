package logger

import (
	"context"
	"github.com/zeromicro/go-zero/core/logx"
	"time"
)

// Logger 扩展日志器
type Logger struct {
	logx.Logger
}

// WithContext 创建日志器
func WithContext(ctx context.Context) *Logger {
	return &Logger{
		Logger: logx.WithContext(ctx),
	}
}

// ========================= 标记方法 =========================

// MarkCritical 标记为关键
func (l *Logger) MarkCritical(value ...any) *Logger {
	return l.mark("critical", value...)
}

// MarkBusiness 标记为业务
func (l *Logger) MarkBusiness(value ...any) *Logger {
	return l.mark("business", value...)
}

// MarkSecurity 标记为安全
func (l *Logger) MarkSecurity(value ...any) *Logger {
	return l.mark("security", value...)
}

// MarkAudit 标记为审计
func (l *Logger) MarkAudit(value ...any) *Logger {
	return l.mark("audit", value...)
}

// ========================= 扩展方法 =========================

// WithField 添加单个字段
func (l *Logger) WithField(key string, value interface{}) *Logger {
	return &Logger{
		Logger: l.Logger.WithFields(logx.Field(key, value)),
	}
}

// WithFields 添加多个字段
func (l *Logger) WithFields(fields ...logx.LogField) *Logger {
	return &Logger{
		Logger: l.Logger.WithFields(fields...),
	}
}

// WithDuration 添加耗时（性能监控常用）
func (l *Logger) WithDuration(d time.Duration) *Logger {
	return &Logger{
		Logger: l.Logger.WithDuration(d),
	}
}

// WithCallerSkip 跳过调用栈（封装时可能需要）
func (l *Logger) WithCallerSkip(skip int) *Logger {
	return &Logger{
		Logger: l.Logger.WithCallerSkip(skip),
	}
}

// mark 通用方法
func (l *Logger) mark(defaultVal any, value ...any) *Logger {
	val := defaultVal
	if len(value) > 0 {
		val = value[0]
	}
	return &Logger{
		Logger: l.Logger.WithFields(logx.Field("mark", val)),
	}
}
