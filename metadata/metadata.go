package metadata

import (
	"context"
	"github.com/spf13/cast"
	"github.com/zeromicro/go-zero/core/logx"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/metadata"
)

const (
	CtxMerchantID   = "x-merchant-id"
	CtxMerchantInfo = "x-merchant-info"
	CtxUserID       = "x-user-id"
	CtxCurrencyCode = "x-currency-code"
	CtxLanguage     = "x-language"
	CtxUserInfo     = "x-user-info"
)

// WithMetadata 上下文数据
func WithMetadata(ctx context.Context, key, val any) context.Context {
	return context.WithValue(ctx, key, val)
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

func GetTraceIDFromCtx(ctx context.Context) string {
	spanCtx := trace.SpanContextFromContext(ctx)
	if spanCtx.HasTraceID() {
		return spanCtx.TraceID().String()
	}
	return ""
}

// WithMerchantIDCurrencyCodeRpcMetadata 设置商户ID 币种到gRPC metadata
func WithMerchantIDCurrencyCodeRpcMetadata(ctx context.Context, merchantID int64, currencyCode string) context.Context {
	if merchantID <= 0 {
		return ctx
	}

	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		md = metadata.New(nil)
	} else {
		md = md.Copy()
	}

	md.Set(CtxMerchantID, cast.ToString(merchantID))
	md.Set(CtxCurrencyCode, currencyCode)
	return metadata.NewOutgoingContext(ctx, md)
}

// GetMerchantIDCurrencyCodeFromRpcMetadata 从gRPC metadata获取商户ID和币种
func GetMerchantIDCurrencyCodeFromRpcMetadata(ctx context.Context) (int64, string) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return 0, ""
	}

	values := md.Get(CtxMerchantID)
	if len(values) == 0 {
		return 0, ""
	}

	merchantID, err := cast.ToInt64E(values[0])
	if err != nil {
		logx.Errorf("GetMerchantIDFromMetadata error: %v", err)
		return 0, ""
	}
	values2 := md.Get(CtxCurrencyCode)
	if len(values2) == 0 {
		return 0, ""
	}
	currencyCode, err := cast.ToStringE(values2[0])
	if err != nil {
		logx.Errorf("GetMerchantIDFromMetadata error: %v", err)
		return 0, ""
	}

	return merchantID, currencyCode
}
