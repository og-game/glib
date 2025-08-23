package metadata

import (
	"context"
	"time"
)

type AsyncTimeout int

const (
	TimeoutShort  AsyncTimeout = 5
	TimeoutNormal AsyncTimeout = 10
	TimeoutLong   AsyncTimeout = 30
)

// ========================= 复制操作 =========================

// CopyCtx 复制 context 的 metadata 到新 context
func CopyCtx(ctx context.Context) context.Context {
	return Extract(ctx).Build()
}

// CopyCtxTimeout 复制 metadata 并设置超时
func CopyCtxTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	return Extract(ctx).Timeout(timeout)
}

// ========================= 异步操作 =========================

// AsyncCtx 创建异步 context（自定义3个常用的超时）
func AsyncCtx(ctx context.Context, t AsyncTimeout) (context.Context, context.CancelFunc) {
	return AsyncCtxTimeout(ctx, time.Duration(t)*time.Second)
}

// AsyncCtxTimeout 创建指定超时的异步 context
func AsyncCtxTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	return Extract(ctx).Timeout(timeout)
}

// AsyncCtxMerchant 创建异步 context 并更新商户信息
func AsyncCtxMerchant(ctx context.Context, merchantID int64, currencyCode string) (context.Context, context.CancelFunc) {
	return Extract(ctx).
		Merchant(merchantID, currencyCode).
		Timeout(10 * time.Second)
}

// ========================= 快速创建 =========================

// CtxBackgroundTimeout 创建带超时的 background context
func CtxBackgroundTimeout(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), timeout)
}

// CtxWithMerchant 快速创建带商户信息的 context
func CtxWithMerchant(merchantID int64, currencyCode string) context.Context {
	return NewBuilder().Merchant(merchantID, currencyCode).Build()
}

// CtxWithMerchantTimeout 创建带商户信息和超时的 context
func CtxWithMerchantTimeout(merchantID int64, currencyCode string, timeout time.Duration) (context.Context, context.CancelFunc) {
	return NewBuilder().Merchant(merchantID, currencyCode).Timeout(timeout)
}
