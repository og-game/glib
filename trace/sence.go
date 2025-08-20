package trace

import (
	"context"
	"fmt"
	"go.opentelemetry.io/otel/attribute"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// ========================= 场景化 Trace 操作 =========================

// EnsureTraceForSchedule 确保定时任务有 Trace
func EnsureTraceForSchedule(ctx context.Context, scheduleName, scheduleID string) (context.Context, string, string) {
	if IsValidTraceContext(ctx) {
		return ctx, GetTraceIDFromCtx(ctx), GetSpanIDFromCtx(ctx)
	}

	ctx, span := EnsureTraceContext(ctx, "scheduler",
		fmt.Sprintf("schedule.%s", scheduleName),
		attribute.String("schedule.id", scheduleID),
		attribute.String("schedule.name", scheduleName),
		attribute.String("trigger.type", "schedule"),
	)

	traceID, spanID := GetTraceIDFromCtx(ctx), GetSpanIDFromCtx(ctx)
	span.End()
	return ctx, traceID, spanID
}

// EnsureTraceForLoop 确保循环任务有 Trace
func EnsureTraceForLoop(ctx context.Context, taskName string, iteration int) (context.Context, string) {
	ctx, span := StartSpanWithService(ctx, "loop-worker",
		fmt.Sprintf("loop.%s", taskName),
		oteltrace.WithSpanKind(oteltrace.SpanKindInternal),
		oteltrace.WithAttributes(
			attribute.String("task.name", taskName),
			attribute.Int("task.iteration", iteration),
			attribute.String("trigger.type", "loop"),
		))

	traceID := GetTraceIDFromCtx(ctx)
	span.End()
	return ctx, traceID
}

// EnsureTraceForWorkflow 确保工作流有 Trace
func EnsureTraceForWorkflow(ctx context.Context, workflowID, workflowType string, memo map[string]string) (context.Context, string, string) {
	// 优先级：
	// 1. 使用 ctx 中已有的 trace
	// 2. 从 memo 恢复 trace
	// 3. 创建新的 trace

	if IsValidTraceContext(ctx) {
		return ctx, GetTraceIDFromCtx(ctx), GetSpanIDFromCtx(ctx)
	}

	if memo != nil {
		if traceID, ok := memo["x-trace-id"]; ok && traceID != "" {
			ctx = RestoreTraceContext(ctx, traceID, memo["x-span-id"])
			if IsValidTraceContext(ctx) {
				return ctx, traceID, memo["x-span-id"]
			}
		}
	}

	ctx, span := EnsureTraceContext(ctx, "workflow",
		fmt.Sprintf("workflow.%s", workflowType),
		attribute.String("workflow.id", workflowID),
		attribute.String("workflow.type", workflowType),
		attribute.String("trigger.type", "workflow"),
	)

	traceID, spanID := GetTraceIDFromCtx(ctx), GetSpanIDFromCtx(ctx)
	span.End()
	return ctx, traceID, spanID
}

// EnsureTraceForHTTP 确保HTTP请求有 Trace（用于非 go-zero 场景）
func EnsureTraceForHTTP(ctx context.Context, method, path string) (context.Context, oteltrace.Span) {
	return EnsureTraceContext(ctx, "http-server",
		fmt.Sprintf("%s %s", method, path),
		attribute.String("http.method", method),
		attribute.String("http.path", path),
		attribute.String("trigger.type", "http"),
	)
}

// EnsureTraceForRPC 确保RPC调用有 Trace（用于非 go-zero 场景）
func EnsureTraceForRPC(ctx context.Context, service, method string) (context.Context, oteltrace.Span) {
	return EnsureTraceContext(ctx, "rpc-server",
		fmt.Sprintf("%s.%s", service, method),
		attribute.String("rpc.service", service),
		attribute.String("rpc.method", method),
		attribute.String("trigger.type", "rpc"),
	)
}

// ========================= MQ 专用操作 =========================

// EnsureTraceForMQConsume 确保MQ消费有 Trace
func EnsureTraceForMQConsume(ctx context.Context, properties map[string]string, topic string) (context.Context, string) {
	// 优先级：
	// 1. 使用 ctx 中已有的 trace
	// 2. 从消息属性恢复 trace
	// 3. 创建新的 trace

	// 如果 ctx 已经有有效的 trace，直接使用
	if IsValidTraceContext(ctx) {
		return ctx, GetTraceIDFromCtx(ctx)
	}

	// 尝试从消息属性恢复 trace
	ctx = ExtractTraceFromMessage(ctx, properties)
	if IsValidTraceContext(ctx) {
		ctx, span := CreateMQConsumerSpan(ctx, topic, 1)
		defer span.End()
		return ctx, GetTraceIDFromCtx(ctx)
	}

	// 创建新的 trace
	ctx, span := EnsureTraceContext(ctx, "mq-consumer",
		fmt.Sprintf("mq.consume.%s", topic),
		attribute.String("mq.topic", topic),
		attribute.String("trigger.type", "mq"),
	)
	traceID := GetTraceIDFromCtx(ctx)
	span.End()
	return ctx, traceID
}

// InjectTraceToMessage 将trace信息注入到消息属性中
func InjectTraceToMessage(ctx context.Context, properties map[string]string) map[string]string {
	if properties == nil {
		properties = make(map[string]string)
	}

	if traceID := GetTraceIDFromCtx(ctx); traceID != "" {
		properties[MessageTraceID] = traceID
		if spanID := GetSpanIDFromCtx(ctx); spanID != "" {
			properties[MessageSpanID] = spanID
		}
	}

	return properties
}

// ExtractTraceFromMessage 从消息属性中提取 trace context
func ExtractTraceFromMessage(ctx context.Context, properties map[string]string) context.Context {
	if properties == nil {
		return ctx
	}

	if traceID, ok := properties[MessageTraceID]; ok && traceID != "" {
		return RestoreTraceContext(ctx, traceID, properties[MessageSpanID])
	}

	return ctx
}

// CreateMQProducerSpan 创建 MQ 生产者 span
func CreateMQProducerSpan(ctx context.Context, topic string, messageCount int) (context.Context, oteltrace.Span) {
	return StartSpanWithService(ctx, "mq-producer",
		fmt.Sprintf("mq.send.%s", topic),
		oteltrace.WithSpanKind(oteltrace.SpanKindProducer),
		oteltrace.WithAttributes(
			attribute.String("messaging.system", "rocketmq"),
			attribute.String("messaging.destination", topic),
			attribute.String("messaging.operation", "send"),
			attribute.Int("messaging.batch.message_count", messageCount),
		))
}

// CreateMQConsumerSpan 创建 MQ 消费者 span
func CreateMQConsumerSpan(ctx context.Context, topic string, messageCount int) (context.Context, oteltrace.Span) {
	return StartSpanWithService(ctx, "mq-consumer",
		fmt.Sprintf("mq.consume.%s", topic),
		oteltrace.WithSpanKind(oteltrace.SpanKindConsumer),
		oteltrace.WithAttributes(
			attribute.String("messaging.system", "rocketmq"),
			attribute.String("messaging.destination", topic),
			attribute.String("messaging.operation", "consume"),
			attribute.Int("messaging.batch.message_count", messageCount),
		))
}
