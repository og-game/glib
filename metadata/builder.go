package metadata

import (
	"context"
	"time"
)

// Builder 用于构建带有 metadata 的 context
type Builder struct {
	merchantID     int64
	currencyCode   string
	merchantUserID string
	skipTenant     bool
	platformID     int64
	userID         int64
	userInfo       any
	merchantInfo   any
	language       string
	traceID        string
	spanID         string
	custom         map[interface{}]interface{}
}

// ========================= 创建 Builder =========================

// NewBuilder 创建空的 builder
func NewBuilder() *Builder {
	return &Builder{
		custom: make(map[interface{}]interface{}),
	}
}

// Extract 从 context 中提取 metadata 创建 builder（核心方法）
func Extract(ctx context.Context) *Builder {
	b := NewBuilder()

	// 提取商户信息
	if val, ok := GetMetadata[int64](ctx, CtxMerchantID); ok {
		b.merchantID = val
	}
	if val, ok := GetMetadata[string](ctx, CtxCurrencyCode); ok {
		b.currencyCode = val
	}
	if val, ok := GetMetadata[string](ctx, CtxMerchantUserID); ok {
		b.merchantUserID = val
	}

	// 提取用户信息
	if val, ok := GetMetadata[int64](ctx, CtxUserID); ok {
		b.userID = val
	}
	if val, ok := GetMetadata[string](ctx, CtxUserInfo); ok {
		b.userInfo = val
	}

	// 提取平台信息
	if val, ok := GetMetadata[int64](ctx, CtxPlatformID); ok {
		b.platformID = val
	}
	if val, ok := GetMetadata[string](ctx, CtxLanguage); ok {
		b.language = val
	}

	// 提取其他
	if val, ok := GetMetadata[bool](ctx, CtxSkipTenant); ok {
		b.skipTenant = val
	}
	if val, ok := GetMetadata[string](ctx, CtxMerchantInfo); ok {
		b.merchantInfo = val
	}

	// 提取 trace
	b.traceID, b.spanID = GetTraceFromCtx(ctx)

	return b
}

// ========================= 设置方法 =========================

// Merchant 设置商户信息
func (b *Builder) Merchant(id int64, currencyCode string) *Builder {
	if id > 0 {
		b.merchantID = id
	}
	if currencyCode != "" {
		b.currencyCode = currencyCode
	}
	return b
}

// UserInfo 设置商户用户
func (b *Builder) UserInfo(userInfo any) *Builder {
	b.userInfo = userInfo
	return b
}

// User 设置用户信息
func (b *Builder) User(id int64, merchantUserID string) *Builder {
	if id > 0 {
		b.userID = id
	}
	if merchantUserID != "" {
		b.merchantUserID = merchantUserID
	}
	return b
}

// Platform 设置平台
func (b *Builder) Platform(id int64) *Builder {
	if id > 0 {
		b.platformID = id
	}
	return b
}

// Language 设置语言
func (b *Builder) Language(lang string) *Builder {
	if lang != "" {
		b.language = lang
	}
	return b
}

// SkipTenant 跳过租户检查
func (b *Builder) SkipTenant() *Builder {
	b.skipTenant = true
	return b
}

// Custom 设置自定义值
func (b *Builder) Custom(key, value interface{}) *Builder {
	b.custom[key] = value
	return b
}

// ========================= 构建方法 =========================

// Build 构建新的 context（从 background 开始）
func (b *Builder) Build() context.Context {
	ctx := context.Background()

	// 按类别添加 metadata
	ctx = b.applyMerchantData(ctx)
	ctx = b.applyUserData(ctx)
	ctx = b.applyPlatformData(ctx)
	ctx = b.applyTraceData(ctx)
	ctx = b.applyCustomData(ctx)

	return ctx
}

// Timeout 构建带超时的 context
func (b *Builder) Timeout(d time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(b.Build(), d)
}

// Deadline 构建带截止时间的 context
func (b *Builder) Deadline(t time.Time) (context.Context, context.CancelFunc) {
	return context.WithDeadline(b.Build(), t)
}

// Cancelable 构建可取消的 context
func (b *Builder) Cancelable() (context.Context, context.CancelFunc) {
	return context.WithCancel(b.Build())
}

// ========================= 内部方法 =========================

func (b *Builder) applyMerchantData(ctx context.Context) context.Context {
	if b.merchantID > 0 {
		ctx = context.WithValue(ctx, CtxMerchantID, b.merchantID)
	}
	if b.currencyCode != "" {
		ctx = context.WithValue(ctx, CtxCurrencyCode, b.currencyCode)
	}
	if b.merchantUserID != "" {
		ctx = context.WithValue(ctx, CtxMerchantUserID, b.merchantUserID)
	}
	if b.merchantInfo != "" {
		ctx = context.WithValue(ctx, CtxMerchantInfo, b.merchantInfo)
	}
	return ctx
}

func (b *Builder) applyUserData(ctx context.Context) context.Context {
	if b.userID > 0 {
		ctx = context.WithValue(ctx, CtxUserID, b.userID)
	}
	if b.userInfo != "" {
		ctx = context.WithValue(ctx, CtxUserInfo, b.userInfo)
	}
	return ctx
}

func (b *Builder) applyPlatformData(ctx context.Context) context.Context {
	if b.platformID > 0 {
		ctx = context.WithValue(ctx, CtxPlatformID, b.platformID)
	}
	if b.language != "" {
		ctx = context.WithValue(ctx, CtxLanguage, b.language)
	}
	if b.skipTenant {
		ctx = context.WithValue(ctx, CtxSkipTenant, true)
	}
	return ctx
}

func (b *Builder) applyTraceData(ctx context.Context) context.Context {
	if b.traceID != "" {
		ctx = context.WithValue(ctx, CtxTraceHeader, b.traceID)
	}
	if b.spanID != "" {
		ctx = context.WithValue(ctx, CtxSpanHeader, b.spanID)
	}
	return ctx
}

func (b *Builder) applyCustomData(ctx context.Context) context.Context {
	for k, v := range b.custom {
		ctx = context.WithValue(ctx, k, v)
	}
	return ctx
}
