package pkg

import (
	"context"
	"sync/atomic"
	"time"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/interceptor"
	"go.temporal.io/sdk/log"
	"go.temporal.io/sdk/workflow"
)

// 简单的指标收集器
type SimpleMetrics struct {
	ActivityCount int64
	WorkflowCount int64
	ErrorCount    int64
}

var metrics = &SimpleMetrics{}

// SimpleInterceptor 简单的拦截器实现
type SimpleInterceptor struct {
	interceptor.InterceptorBase
	logger log.Logger
}

// NewSimpleInterceptor 创建简单拦截器
func NewSimpleInterceptor(logger log.Logger) interceptor.Interceptor {
	return &SimpleInterceptor{logger: logger}
}

// InterceptActivity 拦截活动调用
func (s *SimpleInterceptor) InterceptActivity(
	ctx context.Context,
	next interceptor.ActivityInboundInterceptor,
) interceptor.ActivityInboundInterceptor {
	return &SimpleActivityInterceptor{
		ActivityInboundInterceptorBase: interceptor.ActivityInboundInterceptorBase{},
		logger:                         s.logger,
		next:                           next,
	}
}

// InterceptWorkflow 拦截工作流调用
func (s *SimpleInterceptor) InterceptWorkflow(
	ctx workflow.Context,
	next interceptor.WorkflowInboundInterceptor,
) interceptor.WorkflowInboundInterceptor {
	return &SimpleWorkflowInterceptor{
		WorkflowInboundInterceptorBase: interceptor.WorkflowInboundInterceptorBase{},
		logger:                         workflow.GetLogger(ctx),
		next:                           next,
	}
}

// SimpleActivityInterceptor 活动拦截器实现
type SimpleActivityInterceptor struct {
	interceptor.ActivityInboundInterceptorBase
	logger log.Logger
	next   interceptor.ActivityInboundInterceptor
}

// ExecuteActivity 执行活动拦截
func (a *SimpleActivityInterceptor) ExecuteActivity(
	ctx context.Context,
	in *interceptor.ExecuteActivityInput,
) (interface{}, error) {
	start := time.Now()

	// 原子递增活动计数
	atomic.AddInt64(&metrics.ActivityCount, 1)

	// 获取活动信息
	activityInfo := activity.GetInfo(ctx)
	activityName := activityInfo.ActivityType.Name

	a.logger.Info("活动开始",
		"activity", activityName,
		"workflow_id", activityInfo.WorkflowExecution.ID,
		"activity_id", activityInfo.ActivityID,
	)

	// 执行活动
	result, err := a.next.ExecuteActivity(ctx, in)

	// 记录结果
	duration := time.Since(start)
	if err != nil {
		atomic.AddInt64(&metrics.ErrorCount, 1)
		a.logger.Error("活动失败",
			"activity", activityName,
			"error", err,
			"duration", duration,
		)
	} else {
		a.logger.Info("活动成功",
			"activity", activityName,
			"duration", duration,
		)
	}

	return result, err
}

// SimpleWorkflowInterceptor 工作流拦截器实现
type SimpleWorkflowInterceptor struct {
	interceptor.WorkflowInboundInterceptorBase
	logger log.Logger
	next   interceptor.WorkflowInboundInterceptor
}

// ExecuteWorkflow 执行工作流拦截
func (w *SimpleWorkflowInterceptor) ExecuteWorkflow(
	ctx workflow.Context,
	in *interceptor.ExecuteWorkflowInput,
) (interface{}, error) {
	start := workflow.Now(ctx)

	// 原子递增工作流计数
	atomic.AddInt64(&metrics.WorkflowCount, 1)

	// 获取工作流信息
	workflowInfo := workflow.GetInfo(ctx)
	workflowName := workflowInfo.WorkflowType.Name

	w.logger.Info("工作流开始",
		"workflow", workflowName,
		"workflow_id", workflowInfo.WorkflowExecution.ID,
		"run_id", workflowInfo.WorkflowExecution.RunID,
	)

	// 执行工作流
	result, err := w.next.ExecuteWorkflow(ctx, in)

	// 记录结果
	duration := workflow.Now(ctx).Sub(start)
	if err != nil {
		atomic.AddInt64(&metrics.ErrorCount, 1)
		w.logger.Error("工作流失败",
			"workflow", workflowName,
			"error", err,
			"duration", duration,
		)
	} else {
		w.logger.Info("工作流成功",
			"workflow", workflowName,
			"duration", duration,
		)
	}

	return result, err
}

// 获取简单指标
func GetSimpleMetrics() map[string]int64 {
	return map[string]int64{
		"activities": atomic.LoadInt64(&metrics.ActivityCount),
		"workflows":  atomic.LoadInt64(&metrics.WorkflowCount),
		"errors":     atomic.LoadInt64(&metrics.ErrorCount),
	}
}

// 重置指标
func ResetMetrics() {
	atomic.StoreInt64(&metrics.ActivityCount, 0)
	atomic.StoreInt64(&metrics.WorkflowCount, 0)
	atomic.StoreInt64(&metrics.ErrorCount, 0)
}
