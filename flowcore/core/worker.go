package core

import (
	"context"
	"fmt"
	"github.com/og-game/glib/flowcore/config"
	"github.com/og-game/glib/flowcore/pkg"
	"go.temporal.io/sdk/interceptor"
	"sync"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

// Worker 包含了一个 Temporal worker
type Worker struct {
	worker worker.Worker
	config *config.WorkerConfig
	client client.Client
}

// Manager 管理多个 workers
type Manager struct {
	mu      sync.RWMutex
	workers []*Worker
	running bool
}

var globalManager = &Manager{
	workers: make([]*Worker, 0),
}

// NewWorker 创建一个新的 worker
func NewWorker(c client.Client, cfg *config.WorkerConfig) *Worker {

	// 创建默认拦截器
	logger := pkg.NewDevelopmentLogger("info")
	defaultInterceptor := pkg.NewSimpleInterceptor(logger)

	options := worker.Options{
		MaxConcurrentActivityExecutionSize:     cfg.MaxConcurrentActivities,
		MaxConcurrentWorkflowTaskExecutionSize: cfg.MaxConcurrentWorkflows,
		EnableSessionWorker:                    cfg.EnableSessionWorker,
		StickyScheduleToStartTimeout:           cfg.StickyScheduleToStart,
		Interceptors:                           []interceptor.WorkerInterceptor{defaultInterceptor}, // 默认添加拦截器
	}

	w := worker.New(c, cfg.TaskQueue, options)

	return &Worker{
		worker: w,
		config: cfg,
		client: c,
	}
}

// RegisterWorkflow 注册一个工作流
func (w *Worker) RegisterWorkflow(workflow interface{}) {
	w.worker.RegisterWorkflow(workflow)
}

// RegisterActivity 注册一个活动
func (w *Worker) RegisterActivity(activity interface{}) {
	w.worker.RegisterActivity(activity)
}

// Start 启动 worker
func (w *Worker) Start(ctx context.Context) error {
	go func() {
		fmt.Printf("Starting worker for task queue: %s\n", w.config.TaskQueue)
		err := w.worker.Run(worker.InterruptCh())
		if err != nil {
			fmt.Printf("Worker error for task queue %s: %v\n", w.config.TaskQueue, err)
		}
	}()
	return nil
}

// Stop 停止 worker
func (w *Worker) Stop() {
	w.worker.Stop()
	fmt.Printf("Worker stopped for task queue: %s\n", w.config.TaskQueue)
}

// GetTaskQueue 返回任务队列名称
func (w *Worker) GetTaskQueue() string {
	return w.config.TaskQueue
}

// AddWorker 将 worker 添加到全局管理器
func AddWorker(w *Worker) {
	globalManager.mu.Lock()
	defer globalManager.mu.Unlock()
	globalManager.workers = append(globalManager.workers, w)
	fmt.Printf("Added worker for task queue: %s\n", w.config.TaskQueue)
}

// StartAll 启动所有 workers
func StartAll(ctx context.Context) error {
	globalManager.mu.Lock()
	defer globalManager.mu.Unlock()

	if globalManager.running {
		return fmt.Errorf("workers already running")
	}

	fmt.Printf("Starting %d workers\n", len(globalManager.workers))
	for _, w := range globalManager.workers {
		if err := w.Start(ctx); err != nil {
			return fmt.Errorf("failed to start worker for task queue %s: %w", w.config.TaskQueue, err)
		}
	}

	globalManager.running = true
	return nil
}

// StopAll 停止所有 workers
func StopAll() {
	globalManager.mu.Lock()
	defer globalManager.mu.Unlock()

	if !globalManager.running {
		return
	}

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

// IsRunning 返回 workers 是否正在运行
func IsRunning() bool {
	globalManager.mu.RLock()
	defer globalManager.mu.RUnlock()
	return globalManager.running
}

// GetWorkers 返回所有 workers 的副本
func GetWorkers() []*Worker {
	globalManager.mu.RLock()
	defer globalManager.mu.RUnlock()

	workers := make([]*Worker, len(globalManager.workers))
	copy(workers, globalManager.workers)
	return workers
}

// GetInterceptorMetrics 获取拦截器指标
func GetInterceptorMetrics() map[string]int64 {
	return pkg.GetSimpleMetrics()
}

// ResetInterceptorMetrics 重置拦截器指标
func ResetInterceptorMetrics() {
	pkg.ResetMetrics()
}
