package interceptor

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/og-game/glib/flowcore/logger"
	flowcorepkg "github.com/og-game/glib/flowcore/pkg"
	"github.com/og-game/glib/metadata"
	tracex "github.com/og-game/glib/trace"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	oteltrace "go.opentelemetry.io/otel/trace"
	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/interceptor"
	"go.temporal.io/sdk/log"
	"go.temporal.io/sdk/workflow"
)

// ========================= 配置 =========================

// Config 拦截器配置
type Config struct {
	EnableTrace           bool
	EnableMetrics         bool
	EnableDetailedMetrics bool // 启用详细指标
	Logger                log.Logger
	MetricsInterval       time.Duration     // 指标上报间隔
	CustomAttributes      map[string]string // 自定义属性
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		EnableTrace:           true,
		EnableMetrics:         true,
		EnableDetailedMetrics: false,
		MetricsInterval:       5 * time.Minute,
		CustomAttributes:      make(map[string]string),
	}
}

// ========================= 指标定义 =========================

// Metrics 指标收集器
type Metrics struct {
	// 基础计数
	ActivityCount int64
	WorkflowCount int64
	ErrorCount    int64

	// 成功率统计
	ActivitySuccessCount int64
	ActivityFailureCount int64
	WorkflowSuccessCount int64
	WorkflowFailureCount int64

	// 性能指标
	ActivityDuration    int64 // 累计执行时间(毫秒)
	WorkflowDuration    int64
	MaxActivityDuration int64
	MaxWorkflowDuration int64

	// 重试统计
	ActivityRetryCount int64
	WorkflowRetryCount int64

	// 并发统计
	ConcurrentActivities int64
	ConcurrentWorkflows  int64

	// 详细指标
	ActivityMetrics sync.Map // map[string]*ActivityMetric
	WorkflowMetrics sync.Map // map[string]*WorkflowMetric
}

// ActivityMetric 单个活动的指标
type ActivityMetric struct {
	Name          string
	Count         int64
	SuccessCount  int64
	FailureCount  int64
	TotalDuration int64
	MaxDuration   int64
	MinDuration   int64
	RetryCount    int64
	LastError     string
	LastExecution time.Time
}

// WorkflowMetric 单个工作流的指标
type WorkflowMetric struct {
	Name          string
	Count         int64
	SuccessCount  int64
	FailureCount  int64
	TotalDuration int64
	MaxDuration   int64
	MinDuration   int64
	LastError     string
	LastExecution time.Time
}

// ========================= 统一拦截器 =========================

// UnifiedInterceptor 统一的拦截器，整合 trace、metrics 和监控功能
type UnifiedInterceptor struct {
	interceptor.InterceptorBase
	config               *Config
	metrics              *Metrics
	metricsStopChan      chan struct{}
	metricsStopWaitGroup sync.WaitGroup
}

// NewUnifiedInterceptor 创建统一拦截器
func NewUnifiedInterceptor(cfg *Config) interceptor.Interceptor {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	if cfg.Logger == nil {
		cfg.Logger = logger.NewDevelopmentLogger("info")
	}

	ui := &UnifiedInterceptor{
		config:          cfg,
		metrics:         &Metrics{},
		metricsStopChan: make(chan struct{}),
	}

	// 启动指标上报
	if cfg.EnableMetrics && cfg.MetricsInterval > 0 {
		ui.startMetricsReporter()
	}

	return ui
}

// startMetricsReporter 启动指标定期上报
func (u *UnifiedInterceptor) startMetricsReporter() {
	u.metricsStopWaitGroup.Add(1)
	go func() {
		defer u.metricsStopWaitGroup.Done()
		ticker := time.NewTicker(u.config.MetricsInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				u.reportMetrics()
			case <-u.metricsStopChan:
				return
			}
		}
	}()
}

// reportMetrics 上报指标
func (u *UnifiedInterceptor) reportMetrics() {
	metrics := u.GetMetrics()

	u.config.Logger.Info("Temporal Metrics Report",
		"activities", metrics["activities"],
		"workflows", metrics["workflows"],
		"errors", metrics["errors"],
		"activity_success_rate", u.calculateSuccessRate(metrics["activity_success"], metrics["activity_failure"]),
		"workflow_success_rate", u.calculateSuccessRate(metrics["workflow_success"], metrics["workflow_failure"]),
		"concurrent_activities", metrics["concurrent_activities"],
		"concurrent_workflows", metrics["concurrent_workflows"],
	)

	// 如果启用详细指标，输出每个活动和工作流的统计
	if u.config.EnableDetailedMetrics {
		u.reportDetailedMetrics()
	}
}

// reportDetailedMetrics 上报详细指标
func (u *UnifiedInterceptor) reportDetailedMetrics() {
	// 活动指标
	u.metrics.ActivityMetrics.Range(func(key, value interface{}) bool {
		metric := value.(*ActivityMetric)
		avgDuration := int64(0)
		if metric.Count > 0 {
			avgDuration = metric.TotalDuration / metric.Count
		}

		u.config.Logger.Debug("Activity Metric",
			"name", metric.Name,
			"count", metric.Count,
			"success_rate", u.calculateSuccessRate(metric.SuccessCount, metric.FailureCount),
			"avg_duration_ms", avgDuration,
			"max_duration_ms", metric.MaxDuration,
			"retry_count", metric.RetryCount,
		)
		return true
	})

	// 工作流指标
	u.metrics.WorkflowMetrics.Range(func(key, value interface{}) bool {
		metric := value.(*WorkflowMetric)
		avgDuration := int64(0)
		if metric.Count > 0 {
			avgDuration = metric.TotalDuration / metric.Count
		}

		u.config.Logger.Debug("Workflow Metric",
			"name", metric.Name,
			"count", metric.Count,
			"success_rate", u.calculateSuccessRate(metric.SuccessCount, metric.FailureCount),
			"avg_duration_ms", avgDuration,
			"max_duration_ms", metric.MaxDuration,
		)
		return true
	})
}

// calculateSuccessRate 计算成功率
func (u *UnifiedInterceptor) calculateSuccessRate(success, failure int64) float64 {
	total := success + failure
	if total == 0 {
		return 0
	}
	return float64(success) / float64(total) * 100
}

// InterceptActivity 拦截 Activity 调用
func (u *UnifiedInterceptor) InterceptActivity(
	ctx context.Context,
	next interceptor.ActivityInboundInterceptor,
) interceptor.ActivityInboundInterceptor {
	i := &unifiedActivityInboundInterceptor{
		config:  u.config,
		metrics: u.metrics,
	}
	i.Next = next
	return i
}

// InterceptWorkflow 拦截 Workflow 调用
func (u *UnifiedInterceptor) InterceptWorkflow(
	ctx workflow.Context,
	next interceptor.WorkflowInboundInterceptor,
) interceptor.WorkflowInboundInterceptor {
	i := &unifiedWorkflowInboundInterceptor{
		config:  u.config,
		metrics: u.metrics,
		info:    workflow.GetInfo(ctx),
		logger:  workflow.GetLogger(ctx),
	}
	i.Next = next
	return i
}

// GetMetrics 获取当前指标
func (u *UnifiedInterceptor) GetMetrics() map[string]int64 {
	metrics := map[string]int64{
		"activities":            atomic.LoadInt64(&u.metrics.ActivityCount),
		"workflows":             atomic.LoadInt64(&u.metrics.WorkflowCount),
		"errors":                atomic.LoadInt64(&u.metrics.ErrorCount),
		"activity_success":      atomic.LoadInt64(&u.metrics.ActivitySuccessCount),
		"activity_failure":      atomic.LoadInt64(&u.metrics.ActivityFailureCount),
		"workflow_success":      atomic.LoadInt64(&u.metrics.WorkflowSuccessCount),
		"workflow_failure":      atomic.LoadInt64(&u.metrics.WorkflowFailureCount),
		"activity_duration_ms":  atomic.LoadInt64(&u.metrics.ActivityDuration),
		"workflow_duration_ms":  atomic.LoadInt64(&u.metrics.WorkflowDuration),
		"activity_retries":      atomic.LoadInt64(&u.metrics.ActivityRetryCount),
		"workflow_retries":      atomic.LoadInt64(&u.metrics.WorkflowRetryCount),
		"concurrent_activities": atomic.LoadInt64(&u.metrics.ConcurrentActivities),
		"concurrent_workflows":  atomic.LoadInt64(&u.metrics.ConcurrentWorkflows),
	}
	return metrics
}

// GetDetailedMetrics 获取详细指标
func (u *UnifiedInterceptor) GetDetailedMetrics() (map[string]*ActivityMetric, map[string]*WorkflowMetric) {
	activities := make(map[string]*ActivityMetric)
	workflows := make(map[string]*WorkflowMetric)

	u.metrics.ActivityMetrics.Range(func(key, value interface{}) bool {
		activities[key.(string)] = value.(*ActivityMetric)
		return true
	})

	u.metrics.WorkflowMetrics.Range(func(key, value interface{}) bool {
		workflows[key.(string)] = value.(*WorkflowMetric)
		return true
	})

	return activities, workflows
}

// ResetMetrics 重置指标
func (u *UnifiedInterceptor) ResetMetrics() {
	atomic.StoreInt64(&u.metrics.ActivityCount, 0)
	atomic.StoreInt64(&u.metrics.WorkflowCount, 0)
	atomic.StoreInt64(&u.metrics.ErrorCount, 0)
	atomic.StoreInt64(&u.metrics.ActivitySuccessCount, 0)
	atomic.StoreInt64(&u.metrics.ActivityFailureCount, 0)
	atomic.StoreInt64(&u.metrics.WorkflowSuccessCount, 0)
	atomic.StoreInt64(&u.metrics.WorkflowFailureCount, 0)
	atomic.StoreInt64(&u.metrics.ActivityDuration, 0)
	atomic.StoreInt64(&u.metrics.WorkflowDuration, 0)
	atomic.StoreInt64(&u.metrics.ActivityRetryCount, 0)
	atomic.StoreInt64(&u.metrics.WorkflowRetryCount, 0)

	// 清空详细指标
	u.metrics.ActivityMetrics = sync.Map{}
	u.metrics.WorkflowMetrics = sync.Map{}
}

// Stop 停止拦截器
func (u *UnifiedInterceptor) Stop() {
	close(u.metricsStopChan)
	u.metricsStopWaitGroup.Wait()
}

// ========================= Activity 拦截器 =========================

type unifiedActivityInboundInterceptor struct {
	interceptor.ActivityInboundInterceptorBase
	config  *Config
	metrics *Metrics
}

func (a *unifiedActivityInboundInterceptor) Init(outbound interceptor.ActivityOutboundInterceptor) error {
	i := &unifiedActivityOutboundInterceptor{
		config: a.config,
	}
	i.Next = outbound
	return a.Next.Init(i)
}

func (a *unifiedActivityInboundInterceptor) ExecuteActivity(
	ctx context.Context,
	in *interceptor.ExecuteActivityInput,
) (interface{}, error) {
	// 获取 Activity 信息
	info := activity.GetInfo(ctx)
	activityName := info.ActivityType.Name

	// 增加并发计数
	atomic.AddInt64(&a.metrics.ConcurrentActivities, 1)
	defer atomic.AddInt64(&a.metrics.ConcurrentActivities, -1)

	// Metrics: 记录开始时间
	var start time.Time
	if a.config.EnableMetrics {
		start = time.Now()
		atomic.AddInt64(&a.metrics.ActivityCount, 1)

		// 记录重试
		if info.Attempt > 1 {
			atomic.AddInt64(&a.metrics.ActivityRetryCount, 1)
		}

		a.config.Logger.Debug("Activity started",
			"activity", activityName,
			"workflow_id", info.WorkflowExecution.ID,
			"activity_id", info.ActivityID,
			"attempt", info.Attempt,
		)
	}

	// Trace: 从 Header 恢复 trace context
	var span oteltrace.Span
	if a.config.EnableTrace {
		// 从 Header 中读取 trace 信息
		if header := interceptor.Header(ctx); header != nil {
			if contextData := extractContextFromHeader(header); contextData != nil {
				// 恢复 trace context
				if contextData.TraceID != "" {
					ctx = tracex.RestoreTraceContext(ctx, contextData.TraceID, contextData.SpanID)
				}

				// 恢复商户信息
				if contextData.MerchantID > 0 {
					ctx = metadata.WithMetadata(ctx, metadata.CtxMerchantID, contextData.MerchantID)
					ctx = metadata.WithMetadata(ctx, metadata.CtxCurrencyCode, contextData.CurrencyCode)
				}
				if contextData.MerchantUserID != "" {
					ctx = metadata.WithMetadata(ctx, metadata.CtxMerchantUserID, contextData.MerchantUserID)
				}
				if contextData.UserID > 0 {
					ctx = metadata.WithMetadata(ctx, metadata.CtxUserID, contextData.UserID)
				}

				if len(contextData.Baggage) > 0 {
					ctx = metadata.WithMetadata(ctx, metadata.CtxBaggageInfo, contextData.Baggage)
				}
			}
		}

		// 创建新的 span
		tracer := otel.Tracer("temporal-activity")

		// 添加自定义属性
		attributes := []attribute.KeyValue{
			attribute.String("activity.type", activityName),
			attribute.String("activity.id", info.ActivityID),
			attribute.String("workflow.id", info.WorkflowExecution.ID),
			attribute.String("workflow.run_id", info.WorkflowExecution.RunID),
			attribute.String("task_queue", info.TaskQueue),
			attribute.Int("attempt", int(info.Attempt)),
		}

		// 添加自定义属性
		for k, v := range a.config.CustomAttributes {
			attributes = append(attributes, attribute.String(k, v))
		}

		ctx, span = tracer.Start(ctx,
			fmt.Sprintf("activity.%s", activityName),
			oteltrace.WithSpanKind(oteltrace.SpanKindServer),
			oteltrace.WithAttributes(attributes...))

		defer func() {
			if r := recover(); r != nil {
				span.RecordError(fmt.Errorf("panic: %v", r))
				span.SetStatus(codes.Error, fmt.Sprintf("panic: %v", r))
				span.End()
				panic(r)
			}
			span.End()
		}()
	}

	// 执行 Activity
	result, err := a.Next.ExecuteActivity(ctx, in)

	// 记录执行时间
	duration := time.Since(start).Milliseconds()

	// Trace: 记录错误
	if span != nil {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		} else {
			span.SetStatus(codes.Ok, "success")
		}

		// 添加执行时间属性
		span.SetAttributes(attribute.Int64("duration_ms", duration))
	}

	// Metrics: 记录结束和持续时间
	if a.config.EnableMetrics {
		atomic.AddInt64(&a.metrics.ActivityDuration, duration)

		// 更新最大执行时间
		for {
			current := atomic.LoadInt64(&a.metrics.MaxActivityDuration)
			if duration <= current || atomic.CompareAndSwapInt64(&a.metrics.MaxActivityDuration, current, duration) {
				break
			}
		}

		if err != nil {
			atomic.AddInt64(&a.metrics.ErrorCount, 1)
			atomic.AddInt64(&a.metrics.ActivityFailureCount, 1)

			a.config.Logger.Error("Activity failed",
				"activity", activityName,
				"error", err,
				"duration_ms", duration,
				"attempt", info.Attempt,
			)
		} else {
			atomic.AddInt64(&a.metrics.ActivitySuccessCount, 1)

			a.config.Logger.Debug("Activity completed",
				"activity", activityName,
				"duration_ms", duration,
			)
		}

		// 更新详细指标
		if a.config.EnableDetailedMetrics {
			a.updateActivityMetric(activityName, duration, err, info.Attempt)
		}
	}

	return result, err
}

// updateActivityMetric 更新活动的详细指标
func (a *unifiedActivityInboundInterceptor) updateActivityMetric(name string, duration int64, err error, attempt int32) {
	val, _ := a.metrics.ActivityMetrics.LoadOrStore(name, &ActivityMetric{
		Name:        name,
		MinDuration: duration,
	})

	metric := val.(*ActivityMetric)
	atomic.AddInt64(&metric.Count, 1)
	atomic.AddInt64(&metric.TotalDuration, duration)

	if err != nil {
		atomic.AddInt64(&metric.FailureCount, 1)
		metric.LastError = err.Error()
	} else {
		atomic.AddInt64(&metric.SuccessCount, 1)
	}

	if attempt > 1 {
		atomic.AddInt64(&metric.RetryCount, 1)
	}

	// 更新最大/最小执行时间
	for {
		current := atomic.LoadInt64(&metric.MaxDuration)
		if duration <= current || atomic.CompareAndSwapInt64(&metric.MaxDuration, current, duration) {
			break
		}
	}

	for {
		current := atomic.LoadInt64(&metric.MinDuration)
		if duration >= current || atomic.CompareAndSwapInt64(&metric.MinDuration, current, duration) {
			break
		}
	}

	metric.LastExecution = time.Now()
}

type unifiedActivityOutboundInterceptor struct {
	interceptor.ActivityOutboundInterceptorBase
	config *Config
}

func (a *unifiedActivityOutboundInterceptor) GetLogger(ctx context.Context) log.Logger {
	_logger := a.Next.GetLogger(ctx)

	// 如果有 trace span，增强日志
	if a.config.EnableTrace {
		if span := oteltrace.SpanFromContext(ctx); span != nil && span.SpanContext().IsValid() {
			spanCtx := span.SpanContext()
			return log.With(_logger,
				"trace", spanCtx.TraceID().String(),
				"span", spanCtx.SpanID().String(),
			)
		}
	}
	return _logger
}

// ========================= Workflow 拦截器 =========================

type unifiedWorkflowInboundInterceptor struct {
	interceptor.WorkflowInboundInterceptorBase
	config  *Config
	metrics *Metrics
	info    *workflow.Info
	logger  log.Logger
}

func (w *unifiedWorkflowInboundInterceptor) Init(outbound interceptor.WorkflowOutboundInterceptor) error {
	i := &unifiedWorkflowOutboundInterceptor{
		config: w.config,
		logger: w.logger,
	}
	i.Next = outbound
	return w.Next.Init(i)
}

func (w *unifiedWorkflowInboundInterceptor) ExecuteWorkflow(
	ctx workflow.Context,
	in *interceptor.ExecuteWorkflowInput,
) (interface{}, error) {
	workflowName := w.info.WorkflowType.Name

	// 增加并发计数
	atomic.AddInt64(&w.metrics.ConcurrentWorkflows, 1)
	defer atomic.AddInt64(&w.metrics.ConcurrentWorkflows, -1)

	// Metrics: 记录开始时间
	var start time.Time
	if w.config.EnableMetrics {
		start = workflow.Now(ctx)
		atomic.AddInt64(&w.metrics.WorkflowCount, 1)

		w.logger.Debug("Workflow started",
			"workflow", workflowName,
			"workflow_id", w.info.WorkflowExecution.ID,
			"run_id", w.info.WorkflowExecution.RunID,
			"parent_workflow_id", w.info.ParentWorkflowExecution.ID,
		)
	}

	// Trace: 从 Memo 中提取 trace 信息并存储到 workflow context
	if w.config.EnableTrace && w.info.Memo != nil && w.info.Memo.Fields != nil {
		contextData := flowcorepkg.ExtractContextFromMemo(w.info.Memo)

		// 使用新的辅助函数将信息存储到 workflow context
		if contextData.IsValid() {
			ctx = flowcorepkg.WorkflowContextWithData(ctx, contextData)

			w.logger.Info("Workflow started with trace",
				"workflow_type", workflowName,
				"workflow_id", w.info.WorkflowExecution.ID,
				"trace", contextData.TraceID,
				"merchant_id", contextData.MerchantID)
		}
	}

	// 执行 Workflow
	result, err := w.Next.ExecuteWorkflow(ctx, in)

	// 记录执行时间
	duration := workflow.Now(ctx).Sub(start).Milliseconds()

	// Metrics: 记录结束和持续时间
	if w.config.EnableMetrics {
		atomic.AddInt64(&w.metrics.WorkflowDuration, duration)

		// 更新最大执行时间
		for {
			current := atomic.LoadInt64(&w.metrics.MaxWorkflowDuration)
			if duration <= current || atomic.CompareAndSwapInt64(&w.metrics.MaxWorkflowDuration, current, duration) {
				break
			}
		}

		if err != nil {
			atomic.AddInt64(&w.metrics.ErrorCount, 1)
			atomic.AddInt64(&w.metrics.WorkflowFailureCount, 1)

			w.logger.Error("Workflow failed",
				"workflow", workflowName,
				"error", err,
				"duration_ms", duration,
			)
		} else {
			atomic.AddInt64(&w.metrics.WorkflowSuccessCount, 1)

			w.logger.Debug("Workflow completed",
				"workflow", workflowName,
				"duration_ms", duration,
			)
		}

		// 更新详细指标
		if w.config.EnableDetailedMetrics {
			w.updateWorkflowMetric(workflowName, duration, err)
		}
	}

	return result, err
}

// updateWorkflowMetric 更新工作流的详细指标
func (w *unifiedWorkflowInboundInterceptor) updateWorkflowMetric(name string, duration int64, err error) {
	val, _ := w.metrics.WorkflowMetrics.LoadOrStore(name, &WorkflowMetric{
		Name:        name,
		MinDuration: duration,
	})

	metric := val.(*WorkflowMetric)
	atomic.AddInt64(&metric.Count, 1)
	atomic.AddInt64(&metric.TotalDuration, duration)

	if err != nil {
		atomic.AddInt64(&metric.FailureCount, 1)
		metric.LastError = err.Error()
	} else {
		atomic.AddInt64(&metric.SuccessCount, 1)
	}

	// 更新最大/最小执行时间
	for {
		current := atomic.LoadInt64(&metric.MaxDuration)
		if duration <= current || atomic.CompareAndSwapInt64(&metric.MaxDuration, current, duration) {
			break
		}
	}

	for {
		current := atomic.LoadInt64(&metric.MinDuration)
		if duration >= current || atomic.CompareAndSwapInt64(&metric.MinDuration, current, duration) {
			break
		}
	}

	metric.LastExecution = time.Now()
}

type unifiedWorkflowOutboundInterceptor struct {
	interceptor.WorkflowOutboundInterceptorBase
	config *Config
	logger log.Logger
}

func (w *unifiedWorkflowOutboundInterceptor) ExecuteActivity(
	ctx workflow.Context,
	activityType string,
	args ...interface{},
) workflow.Future {
	// Trace: 将 context 写入 header
	if w.config.EnableTrace {
		if header := interceptor.WorkflowHeader(ctx); header != nil {
			contextData := flowcorepkg.DataFromWorkflowContext(ctx)
			if err := writeContextToHeader(contextData, header); err != nil {
				workflow.GetLogger(ctx).Error("Failed to write context to header", "error", err)
			}
		}
	}

	return w.Next.ExecuteActivity(ctx, activityType, args...)
}

func (w *unifiedWorkflowOutboundInterceptor) ExecuteLocalActivity(
	ctx workflow.Context,
	activityType string,
	args ...interface{},
) workflow.Future {
	// Trace: 将 context 写入 header
	if w.config.EnableTrace {
		if header := interceptor.WorkflowHeader(ctx); header != nil {
			contextData := flowcorepkg.DataFromWorkflowContext(ctx)
			if err := writeContextToHeader(contextData, header); err != nil {
				workflow.GetLogger(ctx).Error("Failed to write context to header", "error", err)
			}
		}
	}

	return w.Next.ExecuteLocalActivity(ctx, activityType, args...)
}

func (w *unifiedWorkflowOutboundInterceptor) ExecuteChildWorkflow(
	ctx workflow.Context,
	childWorkflowType string,
	args ...interface{},
) workflow.ChildWorkflowFuture {
	// Trace: 将 context 写入 header（传递给子工作流）
	if w.config.EnableTrace {
		if header := interceptor.WorkflowHeader(ctx); header != nil {
			contextData := flowcorepkg.DataFromWorkflowContext(ctx)
			if err := writeContextToHeader(contextData, header); err != nil {
				workflow.GetLogger(ctx).Error("Failed to write context to header", "error", err)
			}
		}
	}

	return w.Next.ExecuteChildWorkflow(ctx, childWorkflowType, args...)
}

func (w *unifiedWorkflowOutboundInterceptor) GetLogger(ctx workflow.Context) log.Logger {
	loggerVal := w.Next.GetLogger(ctx)

	// 如果启用了 trace，增强日志
	if w.config.EnableTrace {
		fields := make(map[string]interface{})

		// 使用新的 API 获取 trace 信息
		traceID, spanID := flowcorepkg.GetTraceFromWorkflow(ctx)
		if traceID != "" {
			fields["trace"] = traceID
		}
		if spanID != "" {
			fields["span"] = spanID
		}

		// 添加商户信息
		merchantID, currencyCode := flowcorepkg.GetMerchantFromWorkflow(ctx)
		if merchantID > 0 {
			fields["merchant_id"] = merchantID
			fields["currency_code"] = currencyCode
		}

		// 添加自定义属性到日志
		for k, v := range w.config.CustomAttributes {
			fields[k] = v
		}

		if len(fields) > 0 {
			for k, v := range fields {
				loggerVal = log.With(loggerVal, k, v)
			}
		}
	}

	return loggerVal
}

// ========================= 辅助函数 =========================

const headerKey = "unified-context"

// extractContextFromHeader 从 Header 中提取上下文
func extractContextFromHeader(header map[string]*commonpb.Payload) *flowcorepkg.ContextData {
	payload := header[headerKey]
	if payload == nil {
		return nil
	}

	var contextData flowcorepkg.ContextData
	if err := converter.GetDefaultDataConverter().FromPayload(payload, &contextData); err != nil {
		return nil
	}

	return &contextData
}

// writeContextToHeader 将上下文写入 Header
func writeContextToHeader(contextData *flowcorepkg.ContextData, header map[string]*commonpb.Payload) error {
	if contextData == nil {
		return nil
	}

	payload, err := converter.GetDefaultDataConverter().ToPayload(contextData)
	if err != nil {
		return err
	}

	header[headerKey] = payload
	return nil
}

// ========================= 全局实例管理 =========================

var (
	globalInterceptor *UnifiedInterceptor
	globalMutex       sync.RWMutex
)

// GetGlobalInterceptor 获取全局拦截器实例
func GetGlobalInterceptor() *UnifiedInterceptor {
	globalMutex.RLock()
	if globalInterceptor != nil {
		globalMutex.RUnlock()
		return globalInterceptor
	}
	globalMutex.RUnlock()

	globalMutex.Lock()
	defer globalMutex.Unlock()

	// 双重检查
	if globalInterceptor == nil {
		globalInterceptor = NewUnifiedInterceptor(DefaultConfig()).(*UnifiedInterceptor)
	}
	return globalInterceptor
}

// SetGlobalInterceptor 设置全局拦截器
func SetGlobalInterceptor(interceptor *UnifiedInterceptor) {
	globalMutex.Lock()
	defer globalMutex.Unlock()

	// 停止旧的拦截器
	if globalInterceptor != nil {
		globalInterceptor.Stop()
	}

	globalInterceptor = interceptor
}

// GetGlobalMetrics 获取全局指标
func GetGlobalMetrics() map[string]int64 {
	return GetGlobalInterceptor().GetMetrics()
}

// ResetGlobalMetrics 重置全局指标
func ResetGlobalMetrics() {
	GetGlobalInterceptor().ResetMetrics()
}
