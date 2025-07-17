package metadata

import (
	"context"
	"github.com/spf13/cast"
	"github.com/zeromicro/go-zero/core/logx"
	"go.opentelemetry.io/otel/trace"
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
)

// WithMetadata 上下文数据
func WithMetadata(ctx context.Context, key, val any) context.Context {
	return context.WithValue(ctx, key, val)
}

func WithMerchantIDCurrencyCodeMetadata(ctx context.Context, merchantID int64, currencyCode string) context.Context {
	ctx = context.WithValue(ctx, CtxMerchantID, merchantID)
	ctx = context.WithValue(ctx, CtxCurrencyCode, currencyCode)
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

func GetTraceIDFromCtx(ctx context.Context) string {
	spanCtx := trace.SpanContextFromContext(ctx)
	if spanCtx.HasTraceID() {
		return spanCtx.TraceID().String()
	}
	return ""
}

// WithMerchantIDCurrencyCodeMerchantUserIDRpcMetadata 设置商户ID 币种 商户用户id 到gRPC metadata
func WithMerchantIDCurrencyCodeMerchantUserIDRpcMetadata(ctx context.Context, merchantID int64, currencyCode, merchantUserID string) context.Context {
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
	md.Set(CtxMerchantUserID, merchantUserID)
	return metadata.NewOutgoingContext(ctx, md)
}

// GetMerchantIDCurrencyCodeFromRpcMetadata 从gRPC metadata获取商户ID和币种
func GetMerchantIDCurrencyCodeFromRpcMetadata(ctx context.Context) (merchantID int64, currencyCode string, merchantUserID string) {
	var err error
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return
	}

	values := md.Get(CtxMerchantID)
	if len(values) == 0 {
		return
	}

	merchantID, err = cast.ToInt64E(values[0])
	if err != nil {
		logx.Errorf("Get merchant id from metadata error: %v", err)
		return
	}

	values2 := md.Get(CtxCurrencyCode)
	if len(values2) == 0 {
		return
	}

	currencyCode, err = cast.ToStringE(values2[0])
	if err != nil {
		logx.Errorf("Get currency code from metadata error: %v", err)
		return
	}

	values3 := md.Get(CtxMerchantUserID)
	if len(values3) == 0 {
		return
	}

	merchantUserID, err = cast.ToStringE(values3[0])
	if err != nil {
		logx.Errorf("Get merchant user id from metadata error: %v", err)
		return
	}

	return
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
