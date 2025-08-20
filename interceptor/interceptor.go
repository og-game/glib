package interceptor

import (
	"context"
	"github.com/og-game/glib/metadata"
	tracex "github.com/og-game/glib/trace"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
)

// ====== 服务端拦截器（只获取商户ID） ======

// ServerTenantInterceptor 服务端租户拦截器：从Metadata获取商户ID，写入Context
func ServerTenantInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler) (interface{}, error) {

		// 从Metadata获取商户ID，如果没有就是0
		merchantID, currencyCode, merchantUserID := metadata.GetMerchantIDCurrencyCodeFromRpcMetadata(ctx)

		// 获取 go-zero 已创建的 span（如果存在）
		// go-zero 会自动创建 span，我们只需要添加商户属性
		span := trace.SpanFromContext(ctx)
		if span.IsRecording() {
			// 添加商户信息到现有 span
			span.SetAttributes(
				attribute.Int64("merchant.id", merchantID),
				attribute.String("merchant.currency", currencyCode),
				attribute.String("merchant.merchant_user_id", merchantUserID),
			)
		}

		// 写入Context（即使是0也写入）
		ctx = metadata.WithMerchantIDCurrencyCodeMetadata(ctx, merchantID, currencyCode)
		ctx = metadata.WithMerchantUserIDMetadata(ctx, merchantUserID)
		// 继续处理
		resp, err := handler(ctx, req)
		// 记录错误
		if err != nil {
			// note 这个可以取消因为是go-zero
			tracex.RecordError(span, err)
		}

		return resp, err
	}
}

// ServerTenantStreamInterceptor 服务端流式租户拦截器
func ServerTenantStreamInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo,
		handler grpc.StreamHandler) error {

		// 从Metadata获取商户ID
		ctx := ss.Context()

		// 从Metadata获取商户ID，如果没有就是0
		merchantID, currencyCode, merchantUserID := metadata.GetMerchantIDCurrencyCodeFromRpcMetadata(ctx)

		// 获取 go-zero 已创建的 span（如果存在）
		span := trace.SpanFromContext(ctx)
		if span.IsRecording() {
			// 添加商户信息到现有 span
			span.SetAttributes(
				attribute.Int64("merchant.id", merchantID),
				attribute.String("merchant.currency", currencyCode),
				attribute.String("merchant.merchant_user_id", merchantUserID),
			)
		}

		// 写入Context（即使是0也写入）
		ctx = metadata.WithMerchantIDCurrencyCodeMetadata(ctx, merchantID, currencyCode)
		ctx = metadata.WithMerchantUserIDMetadata(ctx, merchantUserID)
		// 包装ServerStream
		wrappedStream := &wrappedServerStream{
			ServerStream: ss,
			ctx:          ctx,
		}

		err := handler(srv, wrappedStream)
		// 记录错误
		if err != nil {
			// note 这个可以取消因为是go-zero
			tracex.RecordError(span, err)
		}

		return err
	}
}

// wrappedServerStream 包装ServerStream以传递新的Context
type wrappedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedServerStream) Context() context.Context {
	return w.ctx
}

// ====== 客户端拦截器（发送商户ID） ======

// ClientTenantInterceptor 客户端租户拦截器：从Context获取商户ID，写入Metadata
func ClientTenantInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{},
		cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {

		// 从Context获取商户ID并写入Metadata（包括0值）
		merchantID, currencyCode := metadata.GetMerchantIDCurrencyCodeFromCtx(ctx)
		merchantUserID := metadata.GetMerchantUserIDFromCtx(ctx)

		// 获取 go-zero 已创建的 span（如果存在）
		span := trace.SpanFromContext(ctx)
		if span.IsRecording() {
			// 添加商户信息到现有 span
			span.SetAttributes(
				attribute.Int64("merchant.id", merchantID),
				attribute.String("merchant.currency", currencyCode),
				attribute.String("merchant.merchant_user_id", merchantUserID),
			)
		}

		// 设置商户信息到gRPC metadata
		ctx = metadata.WithMerchantIDCurrencyCodeMerchantUserIDRpcMetadata(ctx, merchantID, currencyCode, merchantUserID)

		// note 这个可以取消因为是go-zero
		// 注入trace信息到gRPC metadata
		ctx = metadata.InjectTraceToGRPCMetadata(ctx)

		// 继续调用
		err := invoker(ctx, method, req, reply, cc, opts...)
		// 记录错误
		if err != nil {
			// note 这个可以取消因为是go-zero
			tracex.RecordError(span, err)
		}

		return err
	}
}

// ClientTenantStreamInterceptor 客户端流式租户拦截器
func ClientTenantStreamInterceptor() grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn,
		method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {

		// 从Context获取商户ID并写入Metadata
		merchantID, currencyCode := metadata.GetMerchantIDCurrencyCodeFromCtx(ctx)
		merchantUserID := metadata.GetMerchantUserIDFromCtx(ctx)

		// 获取 go-zero 已创建的 span（如果存在）
		span := trace.SpanFromContext(ctx)
		if span.IsRecording() {
			// 添加商户信息到现有 span
			span.SetAttributes(
				attribute.Int64("merchant.id", merchantID),
				attribute.String("merchant.currency", currencyCode),
				attribute.String("merchant.merchant_user_id", merchantUserID),
			)
		}

		// 设置商户信息到gRPC metadata
		ctx = metadata.WithMerchantIDCurrencyCodeMerchantUserIDRpcMetadata(ctx, merchantID, currencyCode, merchantUserID)

		// note 这个可以取消因为是go-zero
		// 注入trace信息到gRPC metadata
		ctx = metadata.InjectTraceToGRPCMetadata(ctx)

		stream, err := streamer(ctx, desc, cc, method, opts...)
		// 记录错误
		if err != nil {
			// note 这个可以取消因为是go-zero
			tracex.RecordError(span, err)
		}

		return stream, err
	}
}
