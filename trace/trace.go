package trace

import (
	"context"
	"fmt"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"runtime"
	"strings"
)

const (
	// MQ 消息中的 trace 字段
	MessageTraceID = "__TRACE_ID__"
	MessageSpanID  = "__SPAN_ID__"

	// 用于标识 remote context
	MessageTraceParent = "__TRACE_PARENT__"
)

// 使用 OpenTelemetry 标准传播器
var propagator = propagation.NewCompositeTextMapPropagator(
	propagation.TraceContext{},
	propagation.Baggage{},
)

// StartSpan 开始一个新的span
// 与 go-zero 保持一致，使用 OpenTelemetry
func StartSpan(ctx context.Context, operationName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	tracer := otel.Tracer("default-lib")
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
// 与 go-zero/core/trace 的实现完全一致
func GetTraceIDFromCtx(ctx context.Context) string {
	spanCtx := trace.SpanContextFromContext(ctx)
	if spanCtx.HasTraceID() {
		return spanCtx.TraceID().String()
	}
	return ""
}

// GetSpanIDFromCtx 从context中获取span ID
// 与 go-zero/core/trace 的实现完全一致
func GetSpanIDFromCtx(ctx context.Context) string {
	spanCtx := trace.SpanContextFromContext(ctx)
	if spanCtx.HasSpanID() {
		return spanCtx.SpanID().String()
	}
	return ""
}

// GetSpanFromContext 从context中获取当前span
func GetSpanFromContext(ctx context.Context) trace.Span {
	return trace.SpanFromContext(ctx)
}

// EnsureTraceContext 确保context中有trace信息
// 如果有则返回，没有则通过 OpenTelemetry 创建
func EnsureTraceContext(ctx context.Context, operationName string) (context.Context, trace.Span) {
	// 检查是否已经有有效的 span
	if span := trace.SpanFromContext(ctx); span.SpanContext().IsValid() {
		// 已经有 span，创建子 span
		tracer := span.TracerProvider().Tracer("default-lib")
		return tracer.Start(ctx, operationName)
	}

	// 没有 span，创建新的根 span
	tracer := otel.Tracer("default-lib")
	return tracer.Start(ctx, operationName, trace.WithSpanKind(trace.SpanKindServer))
}

// InjectTraceToMessage 将trace信息注入到消息属性中
func InjectTraceToMessage(ctx context.Context, properties map[string]string) map[string]string {
	if properties == nil {
		properties = make(map[string]string)
	}

	// 使用 OpenTelemetry 标准传播器
	carrier := &mapCarrier{m: properties}
	propagator.Inject(ctx, carrier)

	// 同时保存简单的 traceId 和 spanId（用于调试和向后兼容）
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
// 恢复 trace context，继承 traceId，创建新的 span
func ExtractTraceFromMessage(ctx context.Context, properties map[string]string) context.Context {
	if properties == nil {
		// 如果没有属性，返回原 context
		// 调用者需要自己创建新的 trace
		return ctx
	}

	// 使用 OpenTelemetry 标准传播器提取
	carrier := &mapCarrier{m: properties}
	ctx = propagator.Extract(ctx, carrier)

	// 检查是否成功提取到 trace context
	spanCtx := trace.SpanContextFromContext(ctx)
	if !spanCtx.IsValid() {
		if traceIDStr, ok := properties[MessageTraceID]; ok && traceIDStr != "" {
			traceID, err := trace.TraceIDFromHex(traceIDStr)
			if err == nil {
				// 创建 remote span context 注意：这里只设置 TraceID，SpanID 会在创建新 span 时自动生成
				spanCtx = trace.NewSpanContext(trace.SpanContextConfig{
					TraceID:    traceID,
					TraceFlags: trace.FlagsSampled,
					Remote:     true, // 标记为远程 context
				})
				ctx = trace.ContextWithRemoteSpanContext(ctx, spanCtx)
			}
		}
	}
	// 返回 context，调用者负责创建具体的 span
	// 例如：ctx, span := trace.StartSpan(ctx, "mq.consumer.process")
	return ctx
}

// TracerFromContext 返回context中的tracer，如果没有则返回全局tracer
// 与 go-zero 的实现类似
func TracerFromContext(ctx context.Context) trace.Tracer {
	if span := trace.SpanFromContext(ctx); span.SpanContext().IsValid() {
		return span.TracerProvider().Tracer("default-lib")
	}
	return otel.Tracer("default-lib")
}

// RecordError 记录错误到span
func RecordError(span trace.Span, err error) {
	if span != nil && err != nil && span.IsRecording() {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
}

// RecordErrorWithContext 从context中获取span并记录错误
func RecordErrorWithContext(ctx context.Context, err error) {
	if err != nil {
		span := trace.SpanFromContext(ctx)
		if span.IsRecording() {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
	}
}

// SetSpanAttributes 设置span属性
func SetSpanAttributes(span trace.Span, attrs map[string]interface{}) {
	if span == nil || !span.IsRecording() {
		return
	}

	kvs := make([]attribute.KeyValue, 0, len(attrs))
	for k, v := range attrs {
		switch val := v.(type) {
		case string:
			kvs = append(kvs, attribute.String(k, val))
		case int:
			kvs = append(kvs, attribute.Int(k, val))
		case int64:
			kvs = append(kvs, attribute.Int64(k, val))
		case float64:
			kvs = append(kvs, attribute.Float64(k, val))
		case bool:
			kvs = append(kvs, attribute.Bool(k, val))
		default:
			kvs = append(kvs, attribute.String(k, fmt.Sprintf("%v", val)))
		}
	}
	span.SetAttributes(kvs...)
}

// SetSpanAttributesWithContext 从context中获取span并设置属性
func SetSpanAttributesWithContext(ctx context.Context, attrs map[string]interface{}) {
	span := trace.SpanFromContext(ctx)
	SetSpanAttributes(span, attrs)
}

// AddEvent 给span添加事件
func AddEvent(ctx context.Context, name string, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		span.AddEvent(name, trace.WithAttributes(attrs...))
	}
}

// IsValidTraceContext 检查context中是否有有效的trace信息
func IsValidTraceContext(ctx context.Context) bool {
	spanCtx := trace.SpanContextFromContext(ctx)
	return spanCtx.IsValid()
}

// GetCallerInfo 获取调用者信息
func GetCallerInfo(skip int) string {
	pc, file, line, ok := runtime.Caller(skip)
	if !ok {
		return "unknown"
	}

	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return fmt.Sprintf("%s:%d", file, line)
	}

	// 简化文件路径
	if idx := strings.LastIndex(file, "/"); idx != -1 {
		file = file[idx+1:]
	}

	return fmt.Sprintf("%s:%d %s", file, line, fn.Name())
}

// Link 创建span之间的链接（用于异步场景）
func Link(parentCtx context.Context) trace.Link {
	return trace.LinkFromContext(parentCtx)
}

// StartSpanWithLinks 创建带链接的span（用于MQ消费者等场景）
func StartSpanWithLinks(ctx context.Context, operationName string, links []trace.Link) (context.Context, trace.Span) {
	tracer := otel.Tracer("default-lib")
	return tracer.Start(ctx, operationName,
		trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithLinks(links...),
	)
}

// mapCarrier 实现 propagation.TextMapCarrier 接口
type mapCarrier struct {
	m map[string]string
}

func (mc *mapCarrier) Get(key string) string {
	return mc.m[key]
}

func (mc *mapCarrier) Set(key, value string) {
	mc.m[key] = value
}

func (mc *mapCarrier) Keys() []string {
	keys := make([]string, 0, len(mc.m))
	for k := range mc.m {
		keys = append(keys, k)
	}
	return keys
}
