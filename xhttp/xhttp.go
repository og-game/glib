package xhttp

import (
	"context"
	tracex "github.com/og-game/glib/trace"
	"github.com/og-game/glib/xerr"
	"github.com/zeromicro/go-zero/rest/httpx"
	"github.com/zeromicro/x/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"net/http"
)

const (
	BusinessCodeOK = 0
	BusinessMsgOk  = "ok"
)

type BaseResponse[T any] struct {
	Code    int    `json:"code" xml:"code"`
	Message string `json:"message" xml:"message"`
	Data    T      `json:"data,omitempty" xml:"data,omitempty"`
	TraceID string `json:"trace_id,omitempty" xml:"trace_id,omitempty"`
}

// JsonBaseResponseCtx writes v into w with appropriate http status code.
func JsonBaseResponseCtx(ctx context.Context, w http.ResponseWriter, v any) {
	traceId := tracex.GetTraceIDFromCtx(ctx)
	spanID := tracex.GetSpanIDFromCtx(ctx)

	// 设置响应头中的 trace 信息
	if traceId != "" {
		w.Header().Set("X-Trace-Id", traceId)
	}
	if spanID != "" {
		w.Header().Set("X-Span-Id", spanID)
	}

	// 获取 HTTP 状态码
	httpStatus := getHttpStatusFromError(v)

	// 写入响应
	httpx.WriteJsonCtx(ctx, w, httpStatus, wrapBaseResponse(v, traceId))
}

// getHttpStatusFromError 根据错误类型返回对应的 HTTP 状态码
func getHttpStatusFromError(v any) int {
	switch data := v.(type) {
	case *xerr.XErr:
		return mapCodeToHttpStatus(data.Code().Int())
	case *errors.CodeMsg:
		return mapCodeToHttpStatus(data.Code)
	case errors.CodeMsg:
		return mapCodeToHttpStatus(data.Code)
	case *status.Status:
		// gRPC status 有更丰富的错误码映射
		return mapGrpcCodeToHttpStatus(data.Code())
	case error:
		// 普通 error 默认返回 500
		return http.StatusInternalServerError
	default:
		// 成功情况
		return http.StatusOK
	}
}

// mapCodeToHttpStatus 将业务错误码映射到 HTTP 状态码
func mapCodeToHttpStatus(code int) int {
	// 如果错误码本身就是 HTTP 状态码范围（100-599）
	if code >= 100 && code < 600 {
		return code
	}

	// 其他情况返回 200，让客户端通过业务码判断
	return http.StatusOK
}

// mapGrpcCodeToHttpStatus 将 gRPC 错误码映射到 HTTP 状态码
func mapGrpcCodeToHttpStatus(code codes.Code) int {
	switch code {
	case codes.OK:
		return http.StatusOK
	case codes.Canceled:
		return http.StatusRequestTimeout
	case codes.Unknown:
		return http.StatusInternalServerError
	case codes.InvalidArgument:
		return http.StatusBadRequest
	case codes.DeadlineExceeded:
		return http.StatusRequestTimeout
	case codes.NotFound:
		return http.StatusNotFound
	case codes.AlreadyExists:
		return http.StatusConflict
	case codes.PermissionDenied:
		return http.StatusForbidden
	case codes.ResourceExhausted:
		return http.StatusTooManyRequests
	case codes.FailedPrecondition:
		return http.StatusPreconditionFailed
	case codes.Aborted:
		return http.StatusConflict
	case codes.OutOfRange:
		return http.StatusBadRequest
	case codes.Unimplemented:
		return http.StatusNotImplemented
	case codes.Internal:
		return http.StatusInternalServerError
	case codes.Unavailable:
		return http.StatusServiceUnavailable
	case codes.DataLoss:
		return http.StatusInternalServerError
	case codes.Unauthenticated:
		return http.StatusUnauthorized
	default:
		return http.StatusInternalServerError
	}
}

func wrapBaseResponse(v any, traceID string) BaseResponse[any] {
	var resp BaseResponse[any]

	// 设置 trace id
	resp.TraceID = traceID

	switch data := v.(type) {
	case *xerr.XErr:
		resp.Code = data.Code().Int()
		resp.Message = data.Message()
	case *errors.CodeMsg:
		resp.Code = data.Code
		resp.Message = data.Msg
	case errors.CodeMsg:
		resp.Code = data.Code
		resp.Message = data.Msg
	case *status.Status:
		resp.Code = int(data.Code())
		resp.Message = data.Message()
	case error:
		resp.Code = http.StatusInternalServerError
		resp.Message = data.Error()
	default:
		resp.Code = BusinessCodeOK
		resp.Message = BusinessMsgOk
		resp.Data = v
	}

	return resp
}

// ErrorHandler 全局错误处理器（可选）
func ErrorHandler(err error) (int, any) {
	// 可以在这里统一处理错误日志
	return getHttpStatusFromError(err), err
}
