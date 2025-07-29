package trace

import (
	"context"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"log"
)

// Config 简化的配置结构
type Config struct {
	ServiceName    string  `json:",default=default-service"`
	JaegerEndpoint string  `json:",optional"`
	SamplingRate   float64 `json:",default=1.0"`
	EnableStdout   bool    `json:",default=false"`
}

// Init 初始化OpenTelemetry（只在需要创建trace的服务中调用）
func Init(config Config) (*tracesdk.TracerProvider, error) {
	var exporter tracesdk.SpanExporter
	var err error

	// 根据配置选择exporter
	switch {
	case config.JaegerEndpoint != "":
		exporter, err = jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(config.JaegerEndpoint)))
		if err != nil {
			return nil, err
		}
		log.Printf("OpenTelemetry: Using Jaeger at %s", config.JaegerEndpoint)
	case config.EnableStdout:
		exporter, err = stdouttrace.New(stdouttrace.WithPrettyPrint())
		if err != nil {
			return nil, err
		}
		log.Println("OpenTelemetry: Using stdout exporter")
	default:
		log.Println("OpenTelemetry: No exporter, trace propagation only")
		return tracesdk.NewTracerProvider(), nil
	}

	// 创建TracerProvider
	tp := tracesdk.NewTracerProvider(
		tracesdk.WithBatcher(exporter),
		tracesdk.WithSampler(tracesdk.TraceIDRatioBased(config.SamplingRate)),
		tracesdk.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(config.ServiceName),
		)),
	)

	// 设置全局
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	log.Printf("OpenTelemetry: Initialized for %s", config.ServiceName)
	return tp, nil
}

// MustInit 初始化，失败时panic
func MustInit(config Config) *tracesdk.TracerProvider {
	tp, err := Init(config)
	if err != nil {
		log.Fatalf("Failed to initialize OpenTelemetry: %v", err)
	}
	return tp
}

// InitOrNoop 初始化，失败时使用no-op
func InitOrNoop(config Config) *tracesdk.TracerProvider {
	tp, err := Init(config)
	if err != nil {
		log.Printf("OpenTelemetry init failed, using no-op: %v", err)
		return tracesdk.NewTracerProvider()
	}
	return tp
}

// SetupPropagation 只设置传播，不初始化exporter
// 用于只需要传播trace的服务（如普通RPC服务）
func SetupPropagation() {
	otel.SetTextMapPropagator(propagation.TraceContext{})
	otel.SetTracerProvider(tracesdk.NewTracerProvider())
	log.Println("OpenTelemetry: Setup propagation only")
}

// Shutdown 关闭TracerProvider
func Shutdown(tp *tracesdk.TracerProvider) error {
	if tp == nil {
		return nil
	}
	if err := tp.Shutdown(context.Background()); err != nil {
		log.Printf("Error shutting down tracer: %v", err)
		return err
	}
	log.Println("OpenTelemetry: Shutdown complete")
	return nil
}
