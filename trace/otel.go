package trace

import (
	"context"
	"fmt"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"log"
	"time"
)

// Config 配置结构
type Config struct {
	ServiceName    string  `json:",default=default-service"`
	JaegerEndpoint string  `json:",optional"`
	SamplingRate   float64 `json:",default=1.0"`   // 范围是0.0到1.0
	EnableStdout   bool    `json:",default=false"` // 启用控制台输出
}

// Init 初始化OpenTelemetry
func Init(config Config) (*tracesdk.TracerProvider, error) {
	// 构建资源信息
	res := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceNameKey.String(config.ServiceName),
	)

	// 构建采样器
	sampler := tracesdk.TraceIDRatioBased(config.SamplingRate)

	// 选择并创建 exporter
	exporter, exporterName, err := createExporter(config)
	if err != nil {
		return nil, fmt.Errorf("create exporter: %w", err)
	}

	// 构建 TracerProvider 选项
	var opts []tracesdk.TracerProviderOption
	opts = append(opts, tracesdk.WithResource(res))
	opts = append(opts, tracesdk.WithSampler(sampler))

	// 如果有 exporter，添加 batcher
	if exporter != nil {
		opts = append(opts, tracesdk.WithBatcher(exporter))
	}

	// 创建 TracerProvider
	tp := tracesdk.NewTracerProvider(opts...)

	// 设置全局配置
	setupGlobal(tp)

	// 只记录关键信息
	if exporterName != "" {
		log.Printf("OpenTelemetry: %s [%s]", config.ServiceName, exporterName)
	}

	return tp, nil
}

// createExporter 根据配置创建对应的 exporter
func createExporter(config Config) (tracesdk.SpanExporter, string, error) {
	// Jaeger 优先级最高
	if config.JaegerEndpoint != "" {
		exporter, err := jaeger.New(
			jaeger.WithCollectorEndpoint(
				jaeger.WithEndpoint(config.JaegerEndpoint),
			),
		)
		if err != nil {
			return nil, "", err
		}
		return exporter, fmt.Sprintf("jaeger@%s", config.JaegerEndpoint), nil
	}

	// Stdout 用于调试
	if config.EnableStdout {
		exporter, err := stdouttrace.New(
			stdouttrace.WithPrettyPrint(),
		)
		if err != nil {
			return nil, "", err
		}
		return exporter, "stdout", nil
	}

	// 默认：无 exporter，仅内存追踪
	return nil, "memory", nil
}

// setupGlobal 设置全局 TracerProvider 和 Propagator
func setupGlobal(tp *tracesdk.TracerProvider) {
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})
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
		// 创建一个最小的 TracerProvider
		tp = tracesdk.NewTracerProvider()
		setupGlobal(tp)
	}
	return tp
}

// SetupPropagation 只设置传播，不初始化exporter
// 用于只需要传播trace的服务（如普通RPC服务）
func SetupPropagation() {
	tp := tracesdk.NewTracerProvider(
		tracesdk.WithSampler(tracesdk.AlwaysSample()),
	)
	setupGlobal(tp)
}

// Shutdown 关闭TracerProvider
func Shutdown(tp *tracesdk.TracerProvider) error {
	if tp == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := tp.Shutdown(ctx); err != nil {
		return fmt.Errorf("shutdown tracer: %w", err)
	}

	return nil
}
