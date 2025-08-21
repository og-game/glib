package trace

import (
	"context"
	"fmt"
	zerotrace "github.com/zeromicro/go-zero/core/trace"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	oteltrace "go.opentelemetry.io/otel/trace"
	"sync"
)

const (
	// 默认服务名
	DefaultServiceName = "default-lib"
)

var (
	once       sync.Once
	propagator = propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
)

func init() {
	once.Do(func() {
		otel.SetTextMapPropagator(propagator)
	})
}

// ========================= 基础 Trace 操作 =========================

// GetTraceIDFromCtx 从context中获取trace ID（兼容 go-zero）
func GetTraceIDFromCtx(ctx context.Context) string {
	if traceID := zerotrace.TraceIDFromContext(ctx); traceID != "" {
		return traceID
	}
	if spanCtx := oteltrace.SpanContextFromContext(ctx); spanCtx.HasTraceID() {
		return spanCtx.TraceID().String()
	}
	return ""
}

// GetSpanIDFromCtx 从context中获取span ID（兼容 go-zero）
func GetSpanIDFromCtx(ctx context.Context) string {
	if spanID := zerotrace.SpanIDFromContext(ctx); spanID != "" {
		return spanID
	}
	if spanCtx := oteltrace.SpanContextFromContext(ctx); spanCtx.HasSpanID() {
		return spanCtx.SpanID().String()
	}
	return ""
}

// IsValidTraceContext 检查context中是否有有效的trace信息
func IsValidTraceContext(ctx context.Context) bool {
	traceID := GetTraceIDFromCtx(ctx)
	return traceID != "" && traceID != "00000000000000000000000000000000"
}

// RestoreTraceContext 从 traceID 和 spanID 字符串恢复 OpenTelemetry context
// 恢复 trace context（从字符串）
func RestoreTraceContext(ctx context.Context, traceIDStr, spanIDStr string) context.Context {
	if traceIDStr == "" {
		return ctx
	}

	traceID, err := oteltrace.TraceIDFromHex(traceIDStr)
	if err != nil {
		return ctx
	}

	var spanID oteltrace.SpanID
	if spanIDStr != "" {
		spanID, _ = oteltrace.SpanIDFromHex(spanIDStr)
	}

	spanCtx := oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: oteltrace.FlagsSampled,
		Remote:     true,
	})

	return oteltrace.ContextWithRemoteSpanContext(ctx, spanCtx)
}

// CopyContextWithTrace 复制 context 并保留 trace 信息
func CopyContextWithTrace(parentCtx, newCtx context.Context) context.Context {
	if !IsValidTraceContext(parentCtx) {
		return newCtx
	}
	return RestoreTraceContext(newCtx, GetTraceIDFromCtx(parentCtx), GetSpanIDFromCtx(parentCtx))
}

// ========================= Span 创建操作 =========================

// GetTracerFromContext 获取或创建 tracer
func GetTracerFromContext(ctx context.Context, serviceName string) oteltrace.Tracer {
	if span := oteltrace.SpanFromContext(ctx); span.SpanContext().IsValid() {
		return span.TracerProvider().Tracer(serviceName)
	}
	return otel.Tracer(serviceName)
}

// StartSpan 开始一个新的span
func StartSpan(ctx context.Context, operationName string, opts ...oteltrace.SpanStartOption) (context.Context, oteltrace.Span) {
	return StartSpanWithService(ctx, DefaultServiceName, operationName, opts...)
}

// StartSpanWithService 使用指定服务名开始span
func StartSpanWithService(ctx context.Context, serviceName string, operationName string, opts ...oteltrace.SpanStartOption) (context.Context, oteltrace.Span) {
	if serviceName == "" {
		serviceName = DefaultServiceName
	}
	return GetTracerFromContext(ctx, serviceName).Start(ctx, operationName, opts...)
}

// EnsureTraceContext 确保 context 有 trace
func EnsureTraceContext(ctx context.Context, serviceName, operationName string, attrs ...attribute.KeyValue) (context.Context, oteltrace.Span) {
	if serviceName == "" {
		serviceName = DefaultServiceName
	}

	opts := []oteltrace.SpanStartOption{oteltrace.WithAttributes(attrs...)}
	if !IsValidTraceContext(ctx) {
		opts = append(opts, oteltrace.WithSpanKind(oteltrace.SpanKindServer))
	}

	return StartSpanWithService(ctx, serviceName, operationName, opts...)
}

// ========================= Span 辅助操作 =========================

// RecordError 记录错误到 span
func RecordError(span oteltrace.Span, err error) {
	if span != nil && err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
}

// RecordSuccess 记录成功状态到 span
func RecordSuccess(span oteltrace.Span) {
	if span != nil {
		span.SetStatus(codes.Ok, "success")
	}
}

// RecordErrorWithContext 从 context 记录错误
func RecordErrorWithContext(ctx context.Context, err error) {
	RecordError(oteltrace.SpanFromContext(ctx), err)
}

// RecordSuccessWithContext 从 context 记录成功
func RecordSuccessWithContext(ctx context.Context) {
	RecordSuccess(oteltrace.SpanFromContext(ctx))
}

// SetSpanAttributes 设置 span 属性
func SetSpanAttributes(span oteltrace.Span, attrs map[string]interface{}) {
	if span == nil {
		return
	}

	attributes := make([]attribute.KeyValue, 0, len(attrs))
	for k, v := range attrs {
		switch val := v.(type) {
		case string:
			attributes = append(attributes, attribute.String(k, val))
		case int:
			attributes = append(attributes, attribute.Int(k, val))
		case int64:
			attributes = append(attributes, attribute.Int64(k, val))
		case float64:
			attributes = append(attributes, attribute.Float64(k, val))
		case bool:
			attributes = append(attributes, attribute.Bool(k, val))
		default:
			attributes = append(attributes, attribute.String(k, fmt.Sprintf("%v", val)))
		}
	}

	span.SetAttributes(attributes...)
}

// SetSpanAttributesWithContext 批量设置 span 属性
func SetSpanAttributesWithContext(ctx context.Context, attrs map[string]interface{}) {
	SetSpanAttributes(oteltrace.SpanFromContext(ctx), attrs)
}

// AddEvent 添加事件到 span
func AddEvent(span oteltrace.Span, name string, attrs ...attribute.KeyValue) {
	if span != nil {
		span.AddEvent(name, oteltrace.WithAttributes(attrs...))
	}
}

// AddEventWithContext 从 context 添加事件
func AddEventWithContext(ctx context.Context, name string, attrs ...attribute.KeyValue) {
	AddEvent(oteltrace.SpanFromContext(ctx), name, attrs...)
}
