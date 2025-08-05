package trace

import (
	"context"
	"fmt"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const (
	MessageTraceID = "__TRACE_ID__"
	MessageSpanID  = "__SPAN_ID__"
)

// StartSpan 开始一个新的span
func StartSpan(ctx context.Context, operationName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	tracer := otel.Tracer("default")
	return tracer.Start(ctx, operationName, opts...)
}

// StartSpanWithService 使用指定服务名开始span
func StartSpanWithService(ctx context.Context, serviceName, operationName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	if serviceName == "" {
		serviceName = "default"
	}
	tracer := otel.Tracer(serviceName)
	return tracer.Start(ctx, operationName, opts...)
}

// GetTraceIDFromCtx 从context中获取trace ID
func GetTraceIDFromCtx(ctx context.Context) string {
	spanCtx := trace.SpanContextFromContext(ctx)
	if spanCtx.HasTraceID() {
		return spanCtx.TraceID().String()
	}
	return ""
}

// GetSpanIDFromCtx 从context中获取span ID
func GetSpanIDFromCtx(ctx context.Context) string {
	spanCtx := trace.SpanContextFromContext(ctx)
	if spanCtx.HasSpanID() {
		return spanCtx.SpanID().String()
	}
	return ""
}

// RecordError 记录错误到span
func RecordError(span trace.Span, err error) {
	if span != nil && err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
}

// SetSpanAttributes 设置span属性
func SetSpanAttributes(span trace.Span, attrs map[string]interface{}) {
	if span == nil {
		return
	}

	for k, v := range attrs {
		switch val := v.(type) {
		case string:
			span.SetAttributes(attribute.String(k, val))
		case int:
			span.SetAttributes(attribute.Int(k, val))
		case int64:
			span.SetAttributes(attribute.Int64(k, val))
		case float64:
			span.SetAttributes(attribute.Float64(k, val))
		case bool:
			span.SetAttributes(attribute.Bool(k, val))
		default:
			span.SetAttributes(attribute.String(k, fmt.Sprintf("%v", val)))
		}
	}
}

// InjectTraceToMessage 将trace信息注入到消息属性中
func InjectTraceToMessage(ctx context.Context, properties map[string]string) map[string]string {
	if properties == nil {
		properties = make(map[string]string)
	}

	spanCtx := trace.SpanContextFromContext(ctx)
	if spanCtx.HasTraceID() {
		properties[MessageTraceID] = spanCtx.TraceID().String()
	}
	if spanCtx.HasSpanID() {
		properties[MessageSpanID] = spanCtx.SpanID().String()
	}

	return properties
}

// ExtractTraceFromMessage 从消息属性中提取trace信息
func ExtractTraceFromMessage(ctx context.Context, properties map[string]string) context.Context {
	if properties == nil {
		return ctx
	}

	traceIDStr := properties[MessageTraceID]
	spanIDStr := properties[MessageSpanID]

	if traceIDStr == "" {
		return ctx
	}

	// 解析trace ID
	traceID, err := trace.TraceIDFromHex(traceIDStr)
	if err != nil {
		return ctx
	}

	// 解析span ID
	var spanID trace.SpanID
	if spanIDStr != "" {
		spanID, err = trace.SpanIDFromHex(spanIDStr)
		if err != nil {
			return ctx
		}
	}

	// 创建span context
	spanCtx := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
	})

	return trace.ContextWithSpanContext(ctx, spanCtx)
}
