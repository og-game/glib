package metadata

import (
	"context"
	tracex "github.com/og-game/glib/trace"
	"github.com/spf13/cast"
	"github.com/zeromicro/go-zero/core/logx"
	"go.opentelemetry.io/otel/propagation"
	"google.golang.org/grpc/metadata"
)

const (
	CtxMerchantID     = "x-merchant-id"
	CtxMerchantUserID = "x-merchant-user-id"
	CtxMerchantInfo   = "x-merchant-info"
	CtxUserID         = "x-user-id"
	CtxCurrencyCode   = "x-currency-code"
	CtxLanguage       = "x-language"
	CtxUserInfo       = "x-user-info"
	CtxSkipTenant     = "x-skip-tenant" // 跳过租户条件的标记
	CtxPlatformID     = "x-platform-id"
	CtxTraceHeader    = "x-trace-id"
	CtxSpanHeader     = "x-span-id"
	CtxBaggageInfo    = "x-baggage"
)

var (
	// OpenTelemetry propagator
	propagator = propagation.TraceContext{}
)

// WithMetadata 上下文数据
func WithMetadata(ctx context.Context, key, val any) context.Context {
	return context.WithValue(ctx, key, val)
}

func WithMerchantIDCurrencyCodeMetadata(ctx context.Context, merchantID int64, currencyCode string) context.Context {
	if merchantID > 0 {
		ctx = context.WithValue(ctx, CtxMerchantID, merchantID)
	}
	if currencyCode != "" {
		ctx = context.WithValue(ctx, CtxCurrencyCode, currencyCode)
	}
	return ctx
}

func WithMerchantUserIDMetadata(ctx context.Context, merchantUserID string) context.Context {
	ctx = context.WithValue(ctx, CtxMerchantUserID, merchantUserID)
	return ctx
}

// GetMetadataFromCtx 获取上下文数据
func GetMetadataFromCtx(ctx context.Context, key any) any {
	return ctx.Value(key)
}

// GetMetadata 上下文取值
func GetMetadata[T any](ctx context.Context, key any) (T, bool) {
	if val, ok := ctx.Value(key).(T); ok {
		return val, true
	}
	var zero T
	return zero, false
}

func GetMerchantIDFromCtx(ctx context.Context) int64 {
	merchantID, _ := GetMetadata[int64](ctx, CtxMerchantID)
	return merchantID
}

func GetCurrencyCodeFromCtx(ctx context.Context) string {
	currencyCode, _ := GetMetadata[string](ctx, CtxCurrencyCode)
	return currencyCode
}

func GetMerchantUserIDFromCtx(ctx context.Context) string {
	merchantUserID, _ := GetMetadata[string](ctx, CtxMerchantUserID)
	return merchantUserID
}

func GetMerchantIDCurrencyCodeFromCtx(ctx context.Context) (merchantID int64, currencyCode string) {
	merchantID, _ = GetMetadata[int64](ctx, CtxMerchantID)
	currencyCode, _ = GetMetadata[string](ctx, CtxCurrencyCode)
	return
}

func GetTraceFromCtx(ctx context.Context) (traceID, spanID string) {
	return tracex.GetTraceIDFromCtx(ctx), tracex.GetSpanIDFromCtx(ctx)
}

// GetTraceLogger 获取带有trace信息的logger
func GetTraceLogger(ctx context.Context) logx.Logger {
	return logx.WithContext(ctx)
}

// WithMerchantIDCurrencyCodeMerchantUserIDRpcMetadata 设置商户ID 币种 商户用户id 到gRPC metadata
func WithMerchantIDCurrencyCodeMerchantUserIDRpcMetadata(ctx context.Context, merchantID int64, currencyCode, merchantUserID string) context.Context {
	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		md = metadata.New(nil)
	} else {
		md = md.Copy()
	}

	md.Set(CtxMerchantID, cast.ToString(merchantID))
	md.Set(CtxCurrencyCode, currencyCode)
	md.Set(CtxMerchantUserID, merchantUserID)
	return metadata.NewOutgoingContext(ctx, md)
}

// InjectTraceToGRPCMetadata 将OpenTelemetry trace信息注入到gRPC metadata中
func InjectTraceToGRPCMetadata(ctx context.Context) context.Context {
	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		md = metadata.New(nil)
	} else {
		md = md.Copy()
	}

	// 使用OpenTelemetry的标准传播器注入trace信息
	propagator.Inject(ctx, &metadataCarrier{md: md})

	// 同时添加自定义的trace头部（用于兼容现有系统）
	traceID, spanID := GetTraceFromCtx(ctx)

	if traceID != "" {
		md.Set(CtxTraceHeader, traceID)
	}
	if spanID != "" {
		md.Set(CtxSpanHeader, spanID)
	}

	return metadata.NewOutgoingContext(ctx, md)
}

// GetMerchantIDCurrencyCodeFromRpcMetadata 从gRPC metadata获取商户ID和币种
func GetMerchantIDCurrencyCodeFromRpcMetadata(ctx context.Context) (merchantID int64, currencyCode string, merchantUserID string) {
	var err error
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return
	}

	if values := md.Get(CtxMerchantID); len(values) > 0 {
		merchantID, err = cast.ToInt64E(values[0])
		if err != nil {
			logx.Errorf("Get merchant id from metadata error: %v", err)
		}
	}

	if values := md.Get(CtxCurrencyCode); len(values) > 0 {
		currencyCode = values[0]
	}

	if values := md.Get(CtxMerchantUserID); len(values) > 0 {
		merchantUserID = values[0]
	}

	return
}

// ExtractTraceFromGRPCMetadata 从gRPC metadata中提取OpenTelemetry trace信息
func ExtractTraceFromGRPCMetadata(ctx context.Context) context.Context {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ctx
	}

	// 提取业务信息
	if values := md.Get(CtxMerchantID); len(values) > 0 {
		if merchantID, err := cast.ToInt64E(values[0]); err == nil {
			ctx = WithMetadata(ctx, CtxMerchantID, merchantID)
		}
	}

	if values := md.Get(CtxCurrencyCode); len(values) > 0 {
		ctx = WithMetadata(ctx, CtxCurrencyCode, values[0])
	}

	if values := md.Get(CtxMerchantUserID); len(values) > 0 {
		ctx = WithMetadata(ctx, CtxMerchantUserID, values[0])
	}

	// 提取trace信息（如果需要恢复）
	if values := md.Get(CtxTraceHeader); len(values) > 0 {
		traceID := values[0]
		var spanID string
		if spanValues := md.Get(CtxSpanHeader); len(spanValues) > 0 {
			spanID = spanValues[0]
		}
		ctx = tracex.RestoreTraceContext(ctx, traceID, spanID)
	}

	return ctx
}

// WithSkipTenant 跳过租户条件
func WithSkipTenant(ctx context.Context) context.Context {
	return context.WithValue(ctx, CtxSkipTenant, true)
}

// ShouldSkipTenant 检查是否跳过租户条件
func ShouldSkipTenant(ctx context.Context) bool {
	if skip, ok := ctx.Value(CtxSkipTenant).(bool); ok {
		return skip
	}
	return false
}

// metadataCarrier 实现OpenTelemetry的TextMapCarrier接口
type metadataCarrier struct {
	md metadata.MD
}

func (mc *metadataCarrier) Get(key string) string {
	values := mc.md.Get(key)
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func (mc *metadataCarrier) Set(key, value string) {
	mc.md.Set(key, value)
}

func (mc *metadataCarrier) Keys() []string {
	var keys []string
	for k := range mc.md {
		keys = append(keys, k)
	}
	return keys
}
