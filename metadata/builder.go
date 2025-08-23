package metadata

import (
	"context"
	"time"
)

// ContextBuilder 用于构建带有 metadata 的 context
type ContextBuilder struct {
	merchantID     int64
	currencyCode   string
	merchantUserID string
	skipTenant     bool
	platformID     int64
	userID         int64
	userInfo       string
	merchantInfo   string
	language       string
	traceID        string
	spanID         string
	customValues   map[interface{}]interface{}
}

// NewBuilder 创建一个全新的 builder（不需要原始 context）
func NewBuilder() *ContextBuilder {
	return &ContextBuilder{
		customValues: make(map[interface{}]interface{}),
	}
}

// NewContextBuilder 从现有 context 创建 builder
func NewContextBuilder(ctx context.Context) *ContextBuilder {
	builder := &ContextBuilder{
		customValues: make(map[interface{}]interface{}),
	}

	// 提取所有已知的 metadata
	if val, ok := GetMetadata[int64](ctx, CtxMerchantID); ok {
		builder.merchantID = val
	}
	if val, ok := GetMetadata[string](ctx, CtxCurrencyCode); ok {
		builder.currencyCode = val
	}
	if val, ok := GetMetadata[string](ctx, CtxMerchantUserID); ok {
		builder.merchantUserID = val
	}
	if val, ok := GetMetadata[bool](ctx, CtxSkipTenant); ok {
		builder.skipTenant = val
	}
	if val, ok := GetMetadata[int64](ctx, CtxPlatformID); ok {
		builder.platformID = val
	}
	if val, ok := GetMetadata[int64](ctx, CtxUserID); ok {
		builder.userID = val
	}
	if val, ok := GetMetadata[string](ctx, CtxUserInfo); ok {
		builder.userInfo = val
	}
	if val, ok := GetMetadata[string](ctx, CtxMerchantInfo); ok {
		builder.merchantInfo = val
	}
	if val, ok := GetMetadata[string](ctx, CtxLanguage); ok {
		builder.language = val
	}

	// 提取 trace 信息
	builder.traceID, builder.spanID = GetTraceFromCtx(ctx)

	return builder
}

// WithMerchantID 设置商户ID
func (b *ContextBuilder) WithMerchantID(merchantID int64) *ContextBuilder {
	b.merchantID = merchantID
	return b
}

// WithCurrencyCode 设置币种
func (b *ContextBuilder) WithCurrencyCode(currencyCode string) *ContextBuilder {
	b.currencyCode = currencyCode
	return b
}

// WithMerchantUserID 设置商户用户ID
func (b *ContextBuilder) WithMerchantUserID(merchantUserID string) *ContextBuilder {
	b.merchantUserID = merchantUserID
	return b
}

// WithSkipTenant 设置跳过租户
func (b *ContextBuilder) WithSkipTenant(skip bool) *ContextBuilder {
	b.skipTenant = skip
	return b
}

// WithPlatformID 设置平台ID
func (b *ContextBuilder) WithPlatformID(platformID int64) *ContextBuilder {
	b.platformID = platformID
	return b
}

// WithUserID 设置用户ID
func (b *ContextBuilder) WithUserID(userID int64) *ContextBuilder {
	b.userID = userID
	return b
}

// WithCustomValue 设置自定义值
func (b *ContextBuilder) WithCustomValue(key, value interface{}) *ContextBuilder {
	b.customValues[key] = value
	return b
}

// Build 构建 context
func (b *ContextBuilder) Build() context.Context {
	ctx := context.Background()

	// 添加所有 metadata
	if b.merchantID > 0 {
		ctx = context.WithValue(ctx, CtxMerchantID, b.merchantID)
	}
	if b.currencyCode != "" {
		ctx = context.WithValue(ctx, CtxCurrencyCode, b.currencyCode)
	}
	if b.merchantUserID != "" {
		ctx = context.WithValue(ctx, CtxMerchantUserID, b.merchantUserID)
	}
	if b.skipTenant {
		ctx = context.WithValue(ctx, CtxSkipTenant, true)
	}
	if b.platformID > 0 {
		ctx = context.WithValue(ctx, CtxPlatformID, b.platformID)
	}
	if b.userID > 0 {
		ctx = context.WithValue(ctx, CtxUserID, b.userID)
	}
	if b.userInfo != "" {
		ctx = context.WithValue(ctx, CtxUserInfo, b.userInfo)
	}
	if b.merchantInfo != "" {
		ctx = context.WithValue(ctx, CtxMerchantInfo, b.merchantInfo)
	}
	if b.language != "" {
		ctx = context.WithValue(ctx, CtxLanguage, b.language)
	}

	// 添加 trace 信息
	if b.traceID != "" {
		ctx = context.WithValue(ctx, CtxTraceHeader, b.traceID)
	}
	if b.spanID != "" {
		ctx = context.WithValue(ctx, CtxSpanHeader, b.spanID)
	}

	// 添加自定义值
	for k, v := range b.customValues {
		ctx = context.WithValue(ctx, k, v)
	}

	return ctx
}

// BuildWithTimeout 构建带超时的 context
func (b *ContextBuilder) BuildWithTimeout(timeout time.Duration) (context.Context, context.CancelFunc) {
	ctx := b.Build()
	return context.WithTimeout(ctx, timeout)
}

// BuildWithCancel 构建可取消的 context
func (b *ContextBuilder) BuildWithCancel() (context.Context, context.CancelFunc) {
	ctx := b.Build()
	return context.WithCancel(ctx)
}

// BuildWithDeadline 构建带截止时间的 context
func (b *ContextBuilder) BuildWithDeadline(deadline time.Time) (context.Context, context.CancelFunc) {
	ctx := b.Build()
	return context.WithDeadline(ctx, deadline)
}

// CopyContext 快速复制 context 的所有 metadata 到新的 background context
func CopyContext(ctx context.Context) context.Context {
	return NewContextBuilder(ctx).Build()
}

// CopyContextWithTimeout 复制 context 并添加超时
func CopyContextWithTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	return NewContextBuilder(ctx).BuildWithTimeout(timeout)
}

// UpdateContext 从现有 context 更新特定字段并创建新 context
func UpdateContext(ctx context.Context, merchantID int64, currencyCode string, timeout time.Duration) (context.Context, context.CancelFunc) {
	return NewContextBuilder(ctx).
		WithMerchantID(merchantID).
		WithCurrencyCode(currencyCode).
		BuildWithTimeout(timeout)
}

// AsyncContextWithData 为异步任务创建 context，更新商户信息
func AsyncContextWithData(ctx context.Context, merchantID int64, currencyCode string) (context.Context, context.CancelFunc) {
	return UpdateContext(ctx, merchantID, currencyCode, 5*time.Second)
}

// AsyncContext 为异步操作创建 context（默认5秒超时）
func AsyncContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return CopyContextWithTimeout(ctx, 5*time.Second)
}

// AsyncContextWithTimeout 为异步操作创建带自定义超时的 context
func AsyncContextWithTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	return CopyContextWithTimeout(ctx, timeout)
}

// BuildBackgroundCtxWithTimeout 快速创建带超时的 context
func BuildBackgroundCtxWithTimeout(timeout time.Duration) (context.Context, context.CancelFunc) {
	return NewBuilder().BuildWithTimeout(timeout)
}

// BuildBackgroundCtx 创 background context
func BuildBackgroundCtx() context.Context {
	return NewBuilder().Build()
}
