package xerr

// ================================
// 基础错误码定义
// ================================

// 预定义的通用错误码（HTTP 标准错误码）
const (
	OK                 Code = 0   // 成功
	Unknown            Code = -1  // 未知错误
	InvalidArgument    Code = 400 // 参数错误
	Unauthorized       Code = 401 // 未认证
	Forbidden          Code = 403 // 无权限
	NotFound           Code = 404 // 资源不存在
	Conflict           Code = 409 // 资源冲突
	TooManyRequests    Code = 429 // 请求过于频繁
	Internal           Code = 500 // 内部错误
	NotImplemented     Code = 501 // 功能未实现
	ServiceUnavailable Code = 503 // 服务不可用
	Timeout            Code = 504 // 请求超时
)

// 基础错误码消息映射
var (
	UnknownError            = New(Unknown, "unknown error")
	InvalidArgumentError    = New(InvalidArgument, "invalid argument")
	UnauthorizedError       = New(Unauthorized, "unauthorized")
	ForbiddenError          = New(Forbidden, "forbidden")
	NotFoundError           = New(NotFound, "not found")
	ConflictError           = New(Conflict, "conflict")
	TooManyRequestsError    = New(TooManyRequests, "too many requests")
	InternalError           = New(Internal, "internal server error")
	NotImplementedError     = New(NotImplemented, "not implemented")
	ServiceUnavailableError = New(ServiceUnavailable, "service unavailable")
	TimeoutError            = New(Timeout, "timeout")
)
