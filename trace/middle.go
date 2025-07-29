package trace

import (
	"context"
	"fmt"
	"go.opentelemetry.io/otel/codes"
)

// LoggerInterface 定义logger接口，适配不同的日志库
type LoggerInterface interface {
	WithFields(fields ...interface{}) LoggerInterface
	Info(msg string)
	Error(msg string)
	Infof(format string, args ...interface{})
	Errorf(format string, args ...interface{})
}

// MessageView 定义消息接口，适配不同的MQ客户端
type MessageView interface {
	GetMessageId() string
	GetProperties() map[string]string
	GetBody() []byte
}

// MiddlewareTrace 通用trace中间件
type MiddlewareTrace struct {
	serviceName string
}

// NewTraceMiddleware 创建trace中间件
func NewTraceMiddleware(serviceName string) *MiddlewareTrace {
	if serviceName == "" {
		serviceName = "default"
	}
	return &MiddlewareTrace{
		serviceName: serviceName,
	}
}

// WrapMQHandler 包装MQ处理器，加入trace支持
func (m *MiddlewareTrace) WrapMQHandler(
	operationName string,
	handler func(ctx context.Context, messages ...MessageView) (bool, error),
	getLogger func(ctx context.Context) LoggerInterface,
) func(ctx context.Context, messages ...MessageView) (bool, error) {
	return func(ctx context.Context, messages ...MessageView) (bool, error) {
		// 创建根span
		ctx, span := StartSpanWithService(ctx, m.serviceName, fmt.Sprintf("mq.consumer.%s", operationName))
		defer span.End()

		// 设置span属性
		SetSpanAttributes(span, map[string]interface{}{
			"messaging.system":           "rocketmq",
			"messaging.operation":        "receive",
			"messaging.destination_name": operationName,
			"messaging.message_count":    len(messages),
			"service.name":               m.serviceName,
		})

		// 获取带有trace信息的logger
		logger := getLogger(ctx)

		logger.WithFields(
			"operation", operationName,
			"message_count", len(messages),
			"trace_id", GetTraceIDFromCtx(ctx),
		).Info("start processing message batch")

		// 处理每个消息的trace信息
		for i, msg := range messages {
			if msg != nil {
				// 尝试从消息属性中提取trace信息
				properties := msg.GetProperties()
				msgCtx := ExtractTraceFromMessage(ctx, properties)

				// 创建消息处理的子span
				_, msgSpan := StartSpanWithService(msgCtx, m.serviceName, fmt.Sprintf("mq.message.process.%d", i))

				SetSpanAttributes(msgSpan, map[string]interface{}{
					"messaging.message_id":    msg.GetMessageId(),
					"messaging.message_index": i,
				})

				msgSpan.End()

				logger.WithFields(
					"message_id", msg.GetMessageId(),
					"message_index", i,
				).Info("processing message")
			}
		}

		// 调用实际的处理器
		success, err := handler(ctx, messages...)

		if err != nil {
			RecordError(span, err)
			logger.WithFields(
				"operation", operationName,
				"error", err.Error(),
			).Error("message batch processing failed")
		} else {
			span.SetStatus(codes.Ok, "success")
			logger.WithFields(
				"operation", operationName,
				"success", success,
			).Info("message batch processing completed")
		}

		return success, err
	}
}

// WrapHTTPHandler 包装HTTP处理器（可以扩展用于web服务）
func (m *MiddlewareTrace) WrapHTTPHandler(
	operationName string,
	handler func(ctx context.Context) error,
	getLogger func(ctx context.Context) LoggerInterface,
) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		// 创建根span
		ctx, span := StartSpanWithService(ctx, m.serviceName, fmt.Sprintf("http.%s", operationName))
		defer span.End()

		// 设置span属性
		SetSpanAttributes(span, map[string]interface{}{
			"http.operation": operationName,
			"service.name":   m.serviceName,
		})

		logger := getLogger(ctx)
		logger.WithFields(
			"operation", operationName,
			"trace_id", GetTraceIDFromCtx(ctx),
		).Info("start processing HTTP request")

		// 调用处理器
		err := handler(ctx)

		if err != nil {
			RecordError(span, err)
			logger.WithFields(
				"operation", operationName,
				"error", err.Error(),
			).Error("HTTP request processing failed")
		} else {
			span.SetStatus(codes.Ok, "success")
			logger.WithFields(
				"operation", operationName,
			).Info("HTTP request processing completed")
		}

		return err
	}
}

// WrapFunction 包装普通函数，加入trace支持
func (m *MiddlewareTrace) WrapFunction(
	ctx context.Context,
	operationName string,
	fn func(ctx context.Context) error,
	getLogger func(ctx context.Context) LoggerInterface,
) error {
	// 创建span
	ctx, span := StartSpanWithService(ctx, m.serviceName, operationName)
	defer span.End()

	// 设置span属性
	SetSpanAttributes(span, map[string]interface{}{
		"operation":    operationName,
		"service.name": m.serviceName,
	})
	// 获取带有trace信息的
	logger := getLogger(ctx)
	logger.WithFields(
		"operation", operationName,
		"trace_id", GetTraceIDFromCtx(ctx),
	).Infof("start %s", operationName)

	// 执行函数
	err := fn(ctx)

	if err != nil {
		RecordError(span, err)
		logger.WithFields(
			"operation", operationName,
			"error", err.Error(),
		).Errorf("%s failed", operationName)
	} else {
		span.SetStatus(codes.Ok, "success")
		logger.WithFields(
			"operation", operationName,
		).Infof("%s completed", operationName)
	}

	return err
}
