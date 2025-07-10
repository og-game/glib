package interceptor

import (
	"context"
	"github.com/og-game/glib/metadata"
	"google.golang.org/grpc"
)

// ====== 服务端拦截器（只获取商户ID） ======

// ServerTenantInterceptor 服务端租户拦截器：从Metadata获取商户ID，写入Context
func ServerTenantInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler) (interface{}, error) {

		// 从Metadata获取商户ID，如果没有就是0
		merchantID, currencyCode := metadata.GetMerchantIDCurrencyCodeFromRpcMetadata(ctx)

		// 写入Context（即使是0也写入）
		ctx = metadata.WithMerchantIDCurrencyCodeMetadata(ctx, merchantID, currencyCode)
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

		// 从Metadata获取商户ID，如果没有就是0
		merchantID, currencyCode := metadata.GetMerchantIDCurrencyCodeFromRpcMetadata(ctx)

		// 写入Context（即使是0也写入）
		ctx = metadata.WithMerchantIDCurrencyCodeMetadata(ctx, merchantID, currencyCode)

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

		// 从Context获取商户ID并写入Metadata（包括0值）
		merchantID, currencyCode := metadata.GetMerchantIDCurrencyCodeFromCtx(ctx)

		ctx = metadata.WithMerchantIDCurrencyCodeRpcMetadata(ctx, merchantID, currencyCode)

		// 继续调用
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

// ClientTenantStreamInterceptor 客户端流式租户拦截器
func ClientTenantStreamInterceptor() grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn,
		method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {

		// 从Context获取商户ID并写入Metadata
		merchantID, currencyCode := metadata.GetMerchantIDCurrencyCodeFromCtx(ctx)

		ctx = metadata.WithMerchantIDCurrencyCodeRpcMetadata(ctx, merchantID, currencyCode)

		return streamer(ctx, desc, cc, method, opts...)
	}
}
