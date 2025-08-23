package metadata

import (
	"context"
	"time"
)

// ========================= 复制操作 =========================

// Copy 复制 context 的 metadata 到新 context
func Copy(ctx context.Context) context.Context {
	return Extract(ctx).Build()
}

// CopyTimeout 复制 metadata 并设置超时
func CopyTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	return Extract(ctx).Timeout(timeout)
}

// ========================= 异步操作 =========================

// Async 创建异步 context（10秒超时）
func Async(ctx context.Context) (context.Context, context.CancelFunc) {
	return Extract(ctx).Timeout(10 * time.Second)
}

// AsyncTimeout 创建指定超时的异步 context
func AsyncTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	return Extract(ctx).Timeout(timeout)
}

// AsyncMerchant 创建异步 context 并更新商户信息
func AsyncMerchant(ctx context.Context, merchantID int64, currencyCode string) (context.Context, context.CancelFunc) {
	return Extract(ctx).
		Merchant(merchantID, currencyCode).
		Timeout(10 * time.Second)
}

// ========================= 快速创建 =========================

// BackgroundTimeout 创建带超时的 background context
func BackgroundTimeout(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), timeout)
}

// WithMerchant 快速创建带商户信息的 context
func WithMerchant(merchantID int64, currencyCode string) context.Context {
	return NewBuilder().Merchant(merchantID, currencyCode).Build()
}

// WithMerchantTimeout 创建带商户信息和超时的 context
func WithMerchantTimeout(merchantID int64, currencyCode string, timeout time.Duration) (context.Context, context.CancelFunc) {
	return NewBuilder().Merchant(merchantID, currencyCode).Timeout(timeout)
}
