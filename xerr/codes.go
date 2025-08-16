package xerr

// 通用错误码
const (
	OK                    int32 = 0
	ErrUnknown            int32 = -1
	ErrInvalidArgument    int32 = 400
	ErrUnauthorized       int32 = 401
	ErrForbidden          int32 = 403
	ErrNotFound           int32 = 404
	ErrConflict           int32 = 409
	ErrTooManyRequests    int32 = 429
	ErrInternal           int32 = 500
	ErrNotImplemented     int32 = 501
	ErrServiceUnavailable int32 = 503
	ErrTimeout            int32 = 504
)

// 预定义错误
var (
	// 通用错误
	ErrUnknownError = New(ErrUnknown, "unknown error")
	ErrInvalidParam = New(ErrInvalidArgument, "invalid parameter")
	ErrUnauth       = New(ErrUnauthorized, "unauthorized")
	ErrNoPermission = New(ErrForbidden, "no permission")
	ErrDataNotFound = New(ErrNotFound, "data not found")
	ErrDataConflict = New(ErrConflict, "data conflict")
	ErrRateLimit    = New(ErrTooManyRequests, "too many requests")
	ErrServerError  = New(ErrInternal, "internal server error")
	ErrNotImpl      = New(ErrNotImplemented, "not implemented")
	ErrServiceDown  = New(ErrServiceUnavailable, "service unavailable")
	ErrReqTimeout   = New(ErrTimeout, "request timeout")
)
