package xerr

import (
	"errors"
	"fmt"
)

// Code 错误码类型
type Code int

func (c Code) Int() int {
	return int(c)
}

// XErr 错误实现
type XErr struct {
	code    Code   // 错误码
	message string // 错误消息
	cause   error  // 原始错误
}

// Error 实现 error 接口
func (e *XErr) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("[%d] %s: %v", e.code, e.message, e.cause)
	}
	return fmt.Sprintf("[%d] %s", e.code, e.message)
}

// Code 获取错误码
func (e *XErr) Code() Code {
	return e.code
}

// Message 获取错误信息
func (e *XErr) Message() string {
	return e.message
}

// Unwrap 实现 errors.Unwrap 接口，支持 errors.Is 和 errors.As
func (e *XErr) Unwrap() error {
	return e.cause
}

// ================================
// 构造函数
// ================================

// New 创建新错误
func New(code Code, message string) error {
	return &XErr{
		code:    code,
		message: message,
	}
}

// Newf 创建格式化错误
func Newf(code Code, format string, args ...interface{}) error {
	return &XErr{
		code:    code,
		message: fmt.Sprintf(format, args...),
	}
}

// Wrap 包装错误
func Wrap(err error, code Code, message string) error {
	if err == nil {
		return nil
	}
	return &XErr{
		code:    code,
		message: message,
		cause:   err,
	}
}

// Wrapf 包装并格式化错误
func Wrapf(err error, code Code, format string, args ...interface{}) error {
	if err == nil {
		return nil
	}
	return &XErr{
		code:    code,
		message: fmt.Sprintf(format, args...),
		cause:   err,
	}
}

// ================================
// 工具函数
// ================================

// Is 判断错误是否为指定错误码
func Is(err error, code Code) bool {
	if err == nil {
		return false
	}

	var xe *XErr
	if errors.As(err, &xe) {
		return xe.code == code
	}
	return false
}

// GetCode 获取错误码
func GetCode(err error) Code {
	if err == nil {
		return OK
	}

	var xe *XErr
	if errors.As(err, &xe) {
		return xe.code
	}
	return Unknown
}

// GetMessage 获取错误消息
func GetMessage(err error) string {
	if err == nil {
		return ""
	}

	var xe *XErr
	if errors.As(err, &xe) {
		return xe.message
	}
	return err.Error()
}
