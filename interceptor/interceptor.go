package interceptor

import (
	"context"
	"fmt"
	"github.com/og-game/glib/metadata"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"google.golang.org/grpc"
)

var tracer = otel.Tracer("grpc-interceptor")

// ====== 服务端拦截器（只获取商户ID） ======

// ServerTenantInterceptor 服务端租户拦截器：从Metadata获取商户ID，写入Context
func ServerTenantInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler) (interface{}, error) {

		// 从gRPC metadata中提取OpenTelemetry trace信息
		ctx = metadata.ExtractTraceFromGRPCMetadata(ctx)

		// 开始一个新的span
		ctx, span := tracer.Start(ctx, fmt.Sprintf("grpc.server.%s", info.FullMethod))
		defer span.End()

		// 从Metadata获取商户ID，如果没有就是0
		merchantID, currencyCode, merchantUserID := metadata.GetMerchantIDCurrencyCodeFromRpcMetadata(ctx)

		// 设置span属性
		span.SetAttributes(
			attribute.String("rpc.service", info.FullMethod),
			attribute.String("rpc.method", info.FullMethod),
			attribute.String("component", "grpc_server"),
			attribute.Int64("merchant.id", merchantID),
			attribute.String("merchant.currency", currencyCode),
			attribute.String("merchant.user_id", merchantUserID),
		)

		// 写入Context（即使是0也写入）
		ctx = metadata.WithMerchantIDCurrencyCodeMetadata(ctx, merchantID, currencyCode)
		ctx = metadata.WithMerchantUserIDMetadata(ctx, merchantUserID)
		// 继续处理
		return handler(ctx, req)
	}
}

// ServerTenantStreamInterceptor 服务端流式租户拦截器
func ServerTenantStreamInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo,
		handler grpc.StreamHandler) error {

		// 从Metadata获取商户ID
		ctx := ss.Context()

		// 从gRPC metadata中提取OpenTelemetry trace信息
		ctx = metadata.ExtractTraceFromGRPCMetadata(ctx)

		// 开始一个新的span
		ctx, span := tracer.Start(ctx, fmt.Sprintf("grpc.server.stream.%s", info.FullMethod))
		defer span.End()

		// 从Metadata获取商户ID，如果没有就是0
		merchantID, currencyCode, merchantUserID := metadata.GetMerchantIDCurrencyCodeFromRpcMetadata(ctx)

		// 设置span属性
		span.SetAttributes(
			attribute.String("rpc.service", info.FullMethod),
			attribute.String("rpc.method", info.FullMethod),
			attribute.String("component", "grpc_server_stream"),
			attribute.Int64("merchant.id", merchantID),
			attribute.String("merchant.currency", currencyCode),
			attribute.String("merchant.user_id", merchantUserID),
		)

		// 写入Context（即使是0也写入）
		ctx = metadata.WithMerchantIDCurrencyCodeMetadata(ctx, merchantID, currencyCode)
		ctx = metadata.WithMerchantUserIDMetadata(ctx, merchantUserID)
		// 包装ServerStream
		wrappedStream := &wrappedServerStream{
			ServerStream: ss,
			ctx:          ctx,
		}

		return handler(srv, wrappedStream)
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

		// 开始一个客户端span
		ctx, span := tracer.Start(ctx, fmt.Sprintf("grpc.client.%s", method))
		defer span.End()

		// 从Context获取商户ID并写入Metadata（包括0值）
		merchantID, currencyCode := metadata.GetMerchantIDCurrencyCodeFromCtx(ctx)
		merchantUserID := metadata.GetMerchantUserIDFromCtx(ctx)

		// 设置span属性
		span.SetAttributes(
			attribute.String("rpc.service", method),
			attribute.String("rpc.method", method),
			attribute.String("component", "grpc_client"),
			attribute.Int64("merchant.id", merchantID),
			attribute.String("merchant.currency", currencyCode),
			attribute.String("merchant.user_id", merchantUserID),
		)
		// 设置商户信息到gRPC metadata
		ctx = metadata.WithMerchantIDCurrencyCodeMerchantUserIDRpcMetadata(ctx, merchantID, currencyCode, merchantUserID)

		// 注入trace信息到gRPC metadata
		ctx = metadata.InjectTraceToGRPCMetadata(ctx)

		// 继续调用
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

// ClientTenantStreamInterceptor 客户端流式租户拦截器
func ClientTenantStreamInterceptor() grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn,
		method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {

		// 开始一个客户端流式span
		ctx, span := tracer.Start(ctx, fmt.Sprintf("grpc.client.stream.%s", method))
		defer span.End()

		// 从Context获取商户ID并写入Metadata
		merchantID, currencyCode := metadata.GetMerchantIDCurrencyCodeFromCtx(ctx)
		merchantUserID := metadata.GetMerchantUserIDFromCtx(ctx)

		// 设置span属性
		span.SetAttributes(
			attribute.String("rpc.service", method),
			attribute.String("rpc.method", method),
			attribute.String("component", "grpc_client_stream"),
			attribute.Int64("merchant.id", merchantID),
			attribute.String("merchant.currency", currencyCode),
			attribute.String("merchant.user_id", merchantUserID),
		)

		// 设置商户信息到gRPC metadata
		ctx = metadata.WithMerchantIDCurrencyCodeMerchantUserIDRpcMetadata(ctx, merchantID, currencyCode, merchantUserID)

		// 注入trace信息到gRPC metadata
		ctx = metadata.InjectTraceToGRPCMetadata(ctx)

		return streamer(ctx, desc, cc, method, opts...)
	}
}
