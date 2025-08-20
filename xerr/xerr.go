package xerr

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
)

// XErr 自定义错误结构
type XErr struct {
	Code  int32  `json:"code"` // 错误码
	Msg   string `json:"msg"`  // 错误信息
	Cause error  `json:"-"`    // 原始错误
	Stack string `json:"-"`    // 调用栈
}

// Error 实现 error 接口
func (e *XErr) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%d] %s: %v", e.Code, e.Msg, e.Cause)
	}
	return fmt.Sprintf("[%d] %s", e.Code, e.Msg)
}

// Unwrap 实现 errors.Unwrap
func (e *XErr) Unwrap() error {
	return e.Cause
}

// WithCause 添加原因错误
func (e *XErr) WithCause(cause error) *XErr {
	e.Cause = cause
	return e
}

// WithMessage 添加额外的错误信息
func (e *XErr) WithMessage(msg string) *XErr {
	if e.Msg != "" {
		e.Msg = fmt.Sprintf("%s: %s", e.Msg, msg)
	} else {
		e.Msg = msg
	}
	return e
}

// WithMessagef 格式化添加错误信息
func (e *XErr) WithMessagef(format string, args ...interface{}) *XErr {
	return e.WithMessage(fmt.Sprintf(format, args...))
}

// GetCode 获取错误码
func (e *XErr) GetCode() int32 {
	return e.Code
}

// GetMsg 获取错误信息
func (e *XErr) GetMsg() string {
	return e.Msg
}

// GetStack 获取调用栈
func (e *XErr) GetStack() string {
	return e.Stack
}

// New 创建新的错误
func New(code int32, msg string) *XErr {
	return &XErr{
		Code:  code,
		Msg:   msg,
		Stack: getStack(2),
	}
}

// Newf 创建格式化的错误
func Newf(code int32, format string, args ...interface{}) *XErr {
	return &XErr{
		Code:  code,
		Msg:   fmt.Sprintf(format, args...),
		Stack: getStack(2),
	}
}

// Wrap 包装错误
func Wrap(err error, code int32, msg string) *XErr {
	if err == nil {
		return nil
	}
	return &XErr{
		Code:  code,
		Msg:   msg,
		Cause: err,
		Stack: getStack(2),
	}
}

// Wrapf 格式化包装错误
func Wrapf(err error, code int32, format string, args ...interface{}) *XErr {
	if err == nil {
		return nil
	}
	return &XErr{
		Code:  code,
		Msg:   fmt.Sprintf(format, args...),
		Cause: err,
		Stack: getStack(2),
	}
}

// getStack 获取调用栈信息
func getStack(skip int) string {
	var builder strings.Builder
	for i := skip; i < skip+10; i++ {
		pc, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}
		fn := runtime.FuncForPC(pc)
		if fn == nil {
			continue
		}
		// 简化文件路径
		if idx := strings.LastIndex(file, "/"); idx != -1 {
			file = file[idx+1:]
		}
		builder.WriteString(fmt.Sprintf("%s:%d %s()\n", file, line, fn.Name()))
	}
	return builder.String()
}

// Is 判断是否为指定错误码
func Is(err error, code int32) bool {
	if err == nil {
		return false
	}
	var xe *XErr
	ok := errors.As(err, &xe)
	if !ok {
		return false
	}
	return xe.Code == code
}

// Code 获取错误码，如果不是 XErr 则返回 0
func Code(err error) int32 {
	if err == nil {
		return 0
	}
	var xe *XErr
	ok := errors.As(err, &xe)
	if !ok {
		return 0
	}
	return xe.Code
}

// Message 获取错误信息
func Message(err error) string {
	if err == nil {
		return ""
	}
	var xe *XErr
	ok := errors.As(err, &xe)
	if !ok {
		return err.Error()
	}
	return xe.Msg
}
