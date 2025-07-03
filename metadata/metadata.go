package metadata

import (
	"context"
	"go.opentelemetry.io/otel/trace"
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
