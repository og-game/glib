package core

import (
	"context"
	"fmt"
	"github.com/og-game/glib/flowcore/config"
	flowcoreinterceptor "github.com/og-game/glib/flowcore/interceptor"
	sdkinterceptor "go.temporal.io/sdk/interceptor"
	"go.temporal.io/sdk/log"
	"sync"
	"time"

	sdkactivity "go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	sdkworkflow "go.temporal.io/sdk/workflow"
)

// Worker 包含了一个 Temporal worker
type Worker struct {
	worker      worker.Worker
	config      *config.WorkerConfig
	client      client.Client
	interceptor *flowcoreinterceptor.UnifiedInterceptor // 统一拦截器实例
	startTime   time.Time                               // 启动时间
	status      WorkerStatus                            // 工作状态
	mu          sync.RWMutex
}

// WorkerStatus 工作器状态
type WorkerStatus int

const (
	WorkerStatusIdle    WorkerStatus = iota // 空闲
	WorkerStatusRunning                     // 运行中
	WorkerStatusStopped                     // 已停止
	WorkerStatusError                       // 错误
)

// Manager 管理多个 workers
type Manager struct {
	mu                sync.RWMutex
	workers           []*Worker
	workersByQueue    map[string]*Worker // 按任务队列索引
	running           bool
	metricsAggregator *MetricsAggregator // 指标聚合器
}

// MetricsAggregator 指标聚合器
type MetricsAggregator struct {
	mu           sync.RWMutex
	stopChan     chan struct{}
	interval     time.Duration
	logger       log.Logger
	enableReport bool
}

var globalManager = &Manager{
	workers:        make([]*Worker, 0),
	workersByQueue: make(map[string]*Worker),
	metricsAggregator: &MetricsAggregator{
		interval:     5 * time.Minute,
		stopChan:     make(chan struct{}),
		enableReport: true,
	},
}

// WorkerOptions worker 配置选项
type WorkerOptions struct {
	EnableTrace           bool                   // 启用 trace
	EnableMetrics         bool                   // 启用指标收集
	EnableDetailedMetrics bool                   // 启用详细指标
	MetricsInterval       time.Duration          // 指标上报间隔
	Logger                log.Logger             // 自定义日志器
	CustomAttributes      map[string]string      // 自定义属性
	OnError               func(error)            // 错误回调
	OnMetrics             func(map[string]int64) // 指标回调
}

// DefaultWorkerOptions 默认配置
func DefaultWorkerOptions() *WorkerOptions {
	return &WorkerOptions{
		EnableTrace:           true,
		EnableMetrics:         true,
		EnableDetailedMetrics: false,
		MetricsInterval:       5 * time.Minute,
		Logger:                nil, // 将使用默认 logger
		CustomAttributes:      make(map[string]string),
	}
}

// NewWorker 创建一个新的 worker
func NewWorker(c client.Client, cfg *config.WorkerConfig) *Worker {
	return NewWorkerWithOptions(c, cfg, DefaultWorkerOptions())
}

// NewWorkerWithOptions 创建带自定义选项的 worker
func NewWorkerWithOptions(c client.Client, cfg *config.WorkerConfig, opts *WorkerOptions) *Worker {
	if opts == nil {
		opts = DefaultWorkerOptions()
	}

	// 创建统一拦截器配置
	interceptorConfig := &flowcoreinterceptor.Config{
		EnableTrace:           opts.EnableTrace,
		EnableMetrics:         opts.EnableMetrics,
		EnableDetailedMetrics: opts.EnableDetailedMetrics,
		MetricsInterval:       opts.MetricsInterval,
		Logger:                opts.Logger,
		CustomAttributes:      opts.CustomAttributes,
	}
	// 创建统一拦截器
	unifiedInterceptor := flowcoreinterceptor.NewUnifiedInterceptor(interceptorConfig).(*flowcoreinterceptor.UnifiedInterceptor)

	// Worker 选项
	options := worker.Options{
		MaxConcurrentActivityExecutionSize:     cfg.MaxConcurrentActivities,
		MaxConcurrentWorkflowTaskExecutionSize: cfg.MaxConcurrentWorkflows,
		EnableSessionWorker:                    cfg.EnableSessionWorker,
		StickyScheduleToStartTimeout:           cfg.StickyScheduleToStart,
		Interceptors: []sdkinterceptor.WorkerInterceptor{
			unifiedInterceptor,
		},
	}

	// 如果有错误回调，设置到选项中
	if opts.OnError != nil {
		options.OnFatalError = opts.OnError
	}

	w := worker.New(c, cfg.TaskQueue, options)

	return &Worker{
		worker:      w,
		config:      cfg,
		client:      c,
		interceptor: unifiedInterceptor,
		status:      WorkerStatusIdle,
	}
}

// RegisterWorkflow 注册一个工作流
func (w *Worker) RegisterWorkflow(workflow interface{}) {
	w.worker.RegisterWorkflow(workflow)
}

// RegisterWorkflowWithAlias 注册带别名的工作流
func (w *Worker) RegisterWorkflowWithAlias(workflow interface{}, alias string) {
	w.worker.RegisterWorkflowWithOptions(workflow, sdkworkflow.RegisterOptions{
		Name: alias,
	})
}

// RegisterActivity 注册一个活动
func (w *Worker) RegisterActivity(activity interface{}) {
	w.worker.RegisterActivity(activity)
}

// RegisterActivityWithOptions 注册带选项的活动
func (w *Worker) RegisterActivityWithOptions(activity interface{}, options sdkactivity.RegisterOptions) {
	w.worker.RegisterActivityWithOptions(activity, options)
}

// Start 启动 worker
func (w *Worker) Start(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.status == WorkerStatusRunning {
		return fmt.Errorf("worker already running for task queue: %s", w.config.TaskQueue)
	}

	go func() {
		w.mu.Lock()
		w.status = WorkerStatusRunning
		w.startTime = time.Now()
		w.mu.Unlock()

		fmt.Printf("Starting worker for task queue: %s\n", w.config.TaskQueue)
		err := w.worker.Run(worker.InterruptCh())

		w.mu.Lock()
		if err != nil {
			w.status = WorkerStatusError
			fmt.Printf("Worker error for task queue %s: %v\n", w.config.TaskQueue, err)
		} else {
			w.status = WorkerStatusStopped
		}
		w.mu.Unlock()
	}()

	return nil
}

// Stop 停止 worker
func (w *Worker) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.status != WorkerStatusRunning {
		return
	}

	w.worker.Stop()
	w.status = WorkerStatusStopped

	// 停止拦截器
	if w.interceptor != nil {
		w.interceptor.Stop()
	}

	fmt.Printf("Worker stopped for task queue: %s\n", w.config.TaskQueue)
}

// GetTaskQueue 返回任务队列名称
func (w *Worker) GetTaskQueue() string {
	return w.config.TaskQueue
}

// GetStatus 获取工作器状态
func (w *Worker) GetStatus() WorkerStatus {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.status
}

// GetUptime 获取运行时间
func (w *Worker) GetUptime() time.Duration {
	w.mu.RLock()
	defer w.mu.RUnlock()

	if w.status != WorkerStatusRunning {
		return 0
	}
	return time.Since(w.startTime)
}

// GetMetrics 获取当前 worker 的指标
func (w *Worker) GetMetrics() map[string]int64 {
	if w.interceptor != nil {
		return w.interceptor.GetMetrics()
	}
	return nil
}

// GetDetailedMetrics 获取详细指标
func (w *Worker) GetDetailedMetrics() (map[string]*flowcoreinterceptor.ActivityMetric, map[string]*flowcoreinterceptor.WorkflowMetric) {
	if w.interceptor != nil {
		return w.interceptor.GetDetailedMetrics()
	}
	return nil, nil
}

// ResetMetrics 重置当前 worker 的指标
func (w *Worker) ResetMetrics() {
	if w.interceptor != nil {
		w.interceptor.ResetMetrics()
	}
}

// GetInfo 获取 Worker 信息
func (w *Worker) GetInfo() WorkerInfo {
	w.mu.RLock()
	defer w.mu.RUnlock()

	info := WorkerInfo{
		TaskQueue:               w.config.TaskQueue,
		Status:                  w.status,
		StartTime:               w.startTime,
		Uptime:                  w.GetUptime(),
		MaxConcurrentActivities: w.config.MaxConcurrentActivities,
		MaxConcurrentWorkflows:  w.config.MaxConcurrentWorkflows,
	}

	if w.interceptor != nil {
		metrics := w.interceptor.GetMetrics()
		info.ActivityCount = metrics["activities"]
		info.WorkflowCount = metrics["workflows"]
		info.ErrorCount = metrics["errors"]
	}

	return info
}

// WorkerInfo Worker 信息
type WorkerInfo struct {
	TaskQueue               string
	Status                  WorkerStatus
	StartTime               time.Time
	Uptime                  time.Duration
	MaxConcurrentActivities int
	MaxConcurrentWorkflows  int
	ActivityCount           int64
	WorkflowCount           int64
	ErrorCount              int64
}

// ========================= Manager 方法 =========================

// AddWorker 将 worker 添加到全局管理器
func AddWorker(w *Worker) {
	globalManager.mu.Lock()
	defer globalManager.mu.Unlock()

	// 检查是否已存在相同任务队列的 worker
	if _, exists := globalManager.workersByQueue[w.config.TaskQueue]; exists {
		fmt.Printf("worker already exists for task queue: %s\n", w.config.TaskQueue)
		return
	}

	globalManager.workers = append(globalManager.workers, w)
	globalManager.workersByQueue[w.config.TaskQueue] = w

	fmt.Printf("Added worker for task queue: %s\n", w.config.TaskQueue)
	return
}

// RemoveWorker 从管理器中移除 worker
func RemoveWorker(taskQueue string) {
	globalManager.mu.Lock()
	defer globalManager.mu.Unlock()

	_worker, exists := globalManager.workersByQueue[taskQueue]
	if !exists {
		fmt.Printf("worker not found for task queue: %s\n", taskQueue)
		return
	}

	// 停止 worker
	_worker.Stop()

	// 从 workers 列表中移除
	for i, w := range globalManager.workers {
		if w.config.TaskQueue == taskQueue {
			globalManager.workers = append(globalManager.workers[:i], globalManager.workers[i+1:]...)
			break
		}
	}

	// 从索引中移除
	delete(globalManager.workersByQueue, taskQueue)

	fmt.Printf("Removed worker for task queue: %s\n", taskQueue)
	return
}

// StartAll 启动所有 workers
func StartAll(ctx context.Context) error {
	globalManager.mu.Lock()
	defer globalManager.mu.Unlock()

	if globalManager.running {
		return fmt.Errorf("workers already running")
	}

	fmt.Printf("Starting %d workers\n", len(globalManager.workers))

	var firstErr error
	successCount := 0

	for _, w := range globalManager.workers {
		if err := w.Start(ctx); err != nil {
			if firstErr == nil {
				firstErr = err
			}
			fmt.Printf("Failed to start worker for task queue %s: %v\n", w.config.TaskQueue, err)
		} else {
			successCount++
		}
	}

	if successCount > 0 {
		globalManager.running = true

		// 启动指标聚合器
		if globalManager.metricsAggregator.enableReport {
			globalManager.metricsAggregator.Start()
		}
	}

	if firstErr != nil {
		return fmt.Errorf("started %d/%d workers, first error: %w",
			successCount, len(globalManager.workers), firstErr)
	}

	return nil
}

// StopAll 停止所有 workers
func StopAll() {
	globalManager.mu.Lock()
	defer globalManager.mu.Unlock()

	if !globalManager.running {
		return
	}

	// 停止指标聚合器
	globalManager.metricsAggregator.Stop()

	fmt.Printf("Stopping %d workers\n", len(globalManager.workers))
	for _, w := range globalManager.workers {
		w.Stop()
	}

	globalManager.running = false
	fmt.Println("All workers stopped")
}

// GetWorkerCount 返回已注册的 worker 数量
func GetWorkerCount() int {
	globalManager.mu.RLock()
	defer globalManager.mu.RUnlock()
	return len(globalManager.workers)
}

// GetRunningWorkerCount 返回运行中的 worker 数量
func GetRunningWorkerCount() int {
	globalManager.mu.RLock()
	defer globalManager.mu.RUnlock()

	count := 0
	for _, w := range globalManager.workers {
		if w.GetStatus() == WorkerStatusRunning {
			count++
		}
	}
	return count
}

// IsRunning 返回 workers 是否正在运行
func IsRunning() bool {
	globalManager.mu.RLock()
	defer globalManager.mu.RUnlock()
	return globalManager.running
}

// GetWorkersInfo 获取所有 workers 的信息
func GetWorkersInfo() []WorkerInfo {
	globalManager.mu.RLock()
	defer globalManager.mu.RUnlock()

	infos := make([]WorkerInfo, 0, len(globalManager.workers))
	for _, w := range globalManager.workers {
		infos = append(infos, w.GetInfo())
	}
	return infos
}

// GetAllMetrics 获取所有 workers 的合并指标
func GetAllMetrics() map[string]int64 {
	globalManager.mu.RLock()
	defer globalManager.mu.RUnlock()

	totalMetrics := map[string]int64{
		"total_workers":         int64(len(globalManager.workers)),
		"running_workers":       0,
		"activities":            0,
		"workflows":             0,
		"errors":                0,
		"activity_success":      0,
		"activity_failure":      0,
		"workflow_success":      0,
		"workflow_failure":      0,
		"activity_duration_ms":  0,
		"workflow_duration_ms":  0,
		"concurrent_activities": 0,
		"concurrent_workflows":  0,
	}

	for _, w := range globalManager.workers {
		if w.GetStatus() == WorkerStatusRunning {
			totalMetrics["running_workers"]++
		}

		if metrics := w.GetMetrics(); metrics != nil {
			for key, value := range metrics {
				totalMetrics[key] += value
			}
		}
	}

	return totalMetrics
}

// ResetAllMetrics 重置所有 workers 的指标
func ResetAllMetrics() {
	globalManager.mu.Lock()
	defer globalManager.mu.Unlock()

	for _, w := range globalManager.workers {
		w.ResetMetrics()
	}
}

// GetWorkerByTaskQueue 根据任务队列名称获取 worker
func GetWorkerByTaskQueue(taskQueue string) *Worker {
	globalManager.mu.RLock()
	defer globalManager.mu.RUnlock()

	return globalManager.workersByQueue[taskQueue]
}

// SetMetricsAggregator 设置指标聚合器
func SetMetricsAggregator(aggregator *MetricsAggregator) {
	globalManager.mu.Lock()
	defer globalManager.mu.Unlock()

	if globalManager.metricsAggregator != nil {
		globalManager.metricsAggregator.Stop()
	}

	globalManager.metricsAggregator = aggregator
}

// ========================= MetricsAggregator 方法 =========================

// Start 启动指标聚合器
func (m *MetricsAggregator) Start() {
	if m.logger == nil {
		return
	}

	go func() {
		ticker := time.NewTicker(m.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				m.reportAggregatedMetrics()
			case <-m.stopChan:
				return
			}
		}
	}()
}

// Stop 停止指标聚合器
func (m *MetricsAggregator) Stop() {
	select {
	case <-m.stopChan:
		// 已经关闭
	default:
		close(m.stopChan)
	}
}

// reportAggregatedMetrics 上报聚合指标
func (m *MetricsAggregator) reportAggregatedMetrics() {
	metrics := GetAllMetrics()

	if m.logger != nil {
		m.logger.Info("Temporal Workers Metrics Report",
			"total_workers", metrics["total_workers"],
			"running_workers", metrics["running_workers"],
			"total_activities", metrics["activities"],
			"total_workflows", metrics["workflows"],
			"total_errors", metrics["errors"],
			"activity_success_rate", calculateSuccessRate(metrics["activity_success"], metrics["activity_failure"]),
			"workflow_success_rate", calculateSuccessRate(metrics["workflow_success"], metrics["workflow_failure"]),
			"concurrent_activities", metrics["concurrent_activities"],
			"concurrent_workflows", metrics["concurrent_workflows"],
		)
	}
}

// calculateSuccessRate 计算成功率
func calculateSuccessRate(success, failure int64) float64 {
	total := success + failure
	if total == 0 {
		return 0
	}
	return float64(success) / float64(total) * 100
}
