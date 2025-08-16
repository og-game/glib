package trace

import (
	"context"
	"go.opentelemetry.io/otel/trace"
)

// RestoreTraceContext 恢复 trace context（用于 Activity）
// 从 traceID 和 spanID 字符串恢复 OpenTelemetry context
func RestoreTraceContext(ctx context.Context, traceIDStr, spanIDStr string) context.Context {
	if traceIDStr == "" {
		return ctx
	}

	// 解析 trace ID
	traceID, err := trace.TraceIDFromHex(traceIDStr)
	if err != nil {
		// 如果解析失败，返回原 context
		return ctx
	}

	// 解析 span ID（可选）
	var spanID trace.SpanID
	if spanIDStr != "" {
		spanID, _ = trace.SpanIDFromHex(spanIDStr)
	}

	// 创建 span context
	spanCtx := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
		Remote:     true, // 标记为远程 context
	})

	// 将 span context 设置到 context 中
	return trace.ContextWithRemoteSpanContext(ctx, spanCtx)
}
