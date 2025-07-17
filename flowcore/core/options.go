package core

import (
	"github.com/og-game/glib/flowcore/pkg"
	"go.temporal.io/sdk/workflow"
	"time"
)

// ActivityOption 活动选项函数类型
type ActivityOption func(*workflow.ActivityOptions)

// WithScheduleToStartTimeout 设置调度到启动超时
func WithScheduleToStartTimeout(timeout time.Duration) ActivityOption {
	return func(opts *workflow.ActivityOptions) {
		opts.ScheduleToStartTimeout = timeout
	}
}

// WithStartToCloseTimeout 设置启动到关闭超时
func WithStartToCloseTimeout(timeout time.Duration) ActivityOption {
	return func(opts *workflow.ActivityOptions) {
		opts.StartToCloseTimeout = timeout
	}
}

// WithHeartbeatTimeout 设置心跳超时
func WithHeartbeatTimeout(timeout time.Duration) ActivityOption {
	return func(opts *workflow.ActivityOptions) {
		opts.HeartbeatTimeout = timeout
	}
}

// WithRetryPolicy 设置重试策略
func WithRetryPolicy(policyName string) ActivityOption {
	return func(opts *workflow.ActivityOptions) {
		opts.RetryPolicy = pkg.Get(policyName)
	}
}

// WithExecutionTimeout 设置执行超时（常用的简化版本）
func WithExecutionTimeout(timeout time.Duration) ActivityOption {
	return func(opts *workflow.ActivityOptions) {
		opts.StartToCloseTimeout = timeout
	}
}
