package flowcore

import (
	"context"
	"fmt"
	"github.com/og-game/glib/flowcore/config"
	"github.com/og-game/glib/flowcore/core"
	"github.com/og-game/glib/flowcore/pkg"
	enumspb "go.temporal.io/api/enums/v1"
	tclient "go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
	"sync"
)

var (
	initialized bool           // 标记 SDK 是否已初始化
	mu          sync.Mutex     // 互斥锁，用于保护 initialized 变量的并发访问
	sdkConfig   *config.Config // 存储 SDK 的配置
)

// Initialize 初始化 Temporal SDK
func Initialize(ctx context.Context, cfg *config.Config) error {
	mu.Lock()         // 获取锁，防止并发初始化
	defer mu.Unlock() // 确保函数退出时释放锁

	if initialized {
		return fmt.Errorf("temporal SDK already initialized") // 如果已初始化，返回错误
	}

	if cfg == nil {
		cfg = config.DefaultConfig() // 如果配置为空，使用默认配置
	}

	// 验证配置
	if err := validateConfig(cfg); err != nil {
		return fmt.Errorf("invalid configuration: %w", err) // 配置无效时返回错误
	}

	// 初始化重试策略
	pkg.InitializeDefaultPolicies()

	// 创建默认客户端
	_, err := core.GetOrCreateDefault(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err) // 创建客户端失败时返回错误
	}

	sdkConfig = cfg    // 保存传入的配置
	initialized = true // 标记 SDK 为已初始化

	fmt.Println("Temporal SDK initialized successfully")
	return nil
}

// MustInitialize 初始化 SDK，如果失败则 panic
func MustInitialize(ctx context.Context, cfg *config.Config) {
	if err := Initialize(ctx, cfg); err != nil {
		panic(fmt.Sprintf("Failed to initialize Temporal SDK: %v", err)) // 初始化失败时触发 panic
	}
}

// Client 返回默认的 Temporal 客户端实例
func Client(ctx context.Context) tclient.Client {
	if !initialized {
		panic("temporal SDK not initialized - call Initialize() first") // 如果未初始化，触发 panic
	}
	c, err := core.GetOrCreateDefault(ctx, sdkConfig) // 获取或创建默认客户端
	if err != nil {
		panic(fmt.Sprintf("failed to get client: %v", err)) // 获取客户端失败时触发 panic
	}
	return c // 返回客户端实例
}

// RegisterWorkflow 全局注册一个工作流
func RegisterWorkflow(name string, workflow interface{}) {
	core.RegisterWorkflow(name, workflow) // 调用核心包的注册工作流方法
}

// RegisterActivity 全局注册一个活动
func RegisterActivity(name string, activity interface{}) {
	core.RegisterActivity(name, activity) // 调用核心包的注册活动方法
}

// RegisterModule 全局注册一个模块
func RegisterModule(module core.Module) {
	core.RegisterModule(module) // 调用核心包的注册模块方法
}

// NewWorker 创建一个新的工作者 (Worker)
func NewWorker(ctx context.Context, cfg *config.WorkerConfig) *core.Worker {
	if !initialized {
		panic("temporal SDK not initialized - call Initialize() first") // 如果未初始化，触发 panic
	}

	c := Client(ctx)            // 获取默认客户端
	w := core.NewWorker(c, cfg) // 使用客户端和工作者配置创建新的工作者

	// 应用全局注册表中的工作流和活动
	core.GetGlobal().ApplyTo(w)

	core.AddWorker(w) // 将新创建的工作者添加到核心包的工作者列表中
	return w          // 返回工作者实例
}

// StartWorkers 启动所有工作者
func StartWorkers(ctx context.Context) error {
	if !initialized {
		return fmt.Errorf("temporal SDK not initialized") // 如果未初始化，返回错误
	}
	return core.StartAll(ctx) // 调用核心包的启动所有工作者方法
}

// StopWorkers 停止所有工作者
func StopWorkers() {
	core.StopAll() // 调用核心包的停止所有工作者方法
}

// Shutdown 关闭 SDK
func Shutdown() {
	StopWorkers()                                 // 停止所有工作者
	core.CloseAll()                               // 关闭所有客户端连接
	mu.Lock()                                     // 获取锁
	initialized = false                           // 标记 SDK 为未初始化
	mu.Unlock()                                   // 释放锁
	fmt.Println("Temporal SDK shutdown complete") // 打印关闭完成信息
}

// IsInitialized 返回 SDK 是否已初始化
func IsInitialized() bool {
	mu.Lock()          // 获取锁
	defer mu.Unlock()  // 确保函数退出时释放锁
	return initialized // 返回初始化状态
}

// GetRetryPolicy 根据名称返回一个重试策略
func GetRetryPolicy(name string) *temporal.RetryPolicy {
	return pkg.Get(name) // 从 pkg 包中获取指定名称的重试策略
}

// WorkflowOptions 创建标准的工作流选项
func WorkflowOptions(workflowID, taskQueue string) tclient.StartWorkflowOptions {
	if !initialized {
		panic("temporal SDK not initialized") // 如果未初始化，触发 panic
	}

	return tclient.StartWorkflowOptions{
		ID:                       workflowID,                                       // 工作流 ID
		TaskQueue:                taskQueue,                                        // 任务队列名称
		WorkflowExecutionTimeout: sdkConfig.Timeout.WorkflowExecution,              // 工作流执行超时时间
		WorkflowTaskTimeout:      sdkConfig.Timeout.WorkflowTask,                   // 工作流任务超时时间
		WorkflowIDReusePolicy:    enumspb.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE, // 工作流 ID 重用策略：允许重复
	}
}

// WorkflowOptionsWithRetry 创建带有自定义重试策略的工作流选项
func WorkflowOptionsWithRetry(workflowID, taskQueue string, retryPolicy *temporal.RetryPolicy) tclient.StartWorkflowOptions {
	options := WorkflowOptions(workflowID, taskQueue) // 获取标准工作流选项
	options.RetryPolicy = retryPolicy                 // 设置自定义重试策略
	return options                                    // 返回带有重试策略的选项
}

// ActivityOptions 创建标准的活动选项
func ActivityOptions(options ...core.ActivityOption) workflow.ActivityOptions {
	if !initialized {
		panic("temporal SDK not initialized") // 如果未初始化，触发 panic
	}

	// 默认选项
	opts := workflow.ActivityOptions{
		ScheduleToStartTimeout: sdkConfig.Timeout.ActivityStart,     // 调度到启动的超时时间
		StartToCloseTimeout:    sdkConfig.Timeout.ActivityExecution, // 启动到关闭的超时时间
		HeartbeatTimeout:       sdkConfig.Timeout.ActivityHeartbeat, // 心跳超时时间
		RetryPolicy:            pkg.Get(pkg.Standard),               // 使用标准重试策略
	}

	// 应用自定义选项
	for _, option := range options {
		option(&opts)
	}

	return opts

}

// ActivityOptionsWithRetry 创建带有自定义重试策略的活动选项
func ActivityOptionsWithRetry(retryPolicyName string, options ...core.ActivityOption) workflow.ActivityOptions {
	opts := ActivityOptions(options...)         // 获取标准活动选项
	opts.RetryPolicy = pkg.Get(retryPolicyName) // 设置指定名称的重试策略
	return opts                                 // 返回带有重试策略的选项
}

// CriticalActivityOptions 创建关键活动的活动选项
func CriticalActivityOptions(options ...core.ActivityOption) workflow.ActivityOptions {
	opts := ActivityOptions(options...)      // 获取标准活动选项
	opts.RetryPolicy = pkg.Get(pkg.Critical) // 使用关键重试策略
	return opts                              // 返回带有关键重试策略的选项
}

// LocalActivityOptions 创建标准的本地活动选项
func LocalActivityOptions() workflow.LocalActivityOptions {
	if !initialized {
		panic("temporal SDK not initialized") // 如果未初始化，触发 panic
	}

	return workflow.LocalActivityOptions{
		ScheduleToCloseTimeout: sdkConfig.Timeout.ActivityExecution, // 调度到关闭的超时时间
		RetryPolicy:            pkg.Get(pkg.Standard),               // 使用标准重试策略
	}
}

// GetConfig 返回当前配置
func GetConfig() *config.Config {
	if !initialized {
		return nil // 如果未初始化，返回 nil
	}
	return sdkConfig // 返回保存的配置
}

// GetStats 返回 SDK 统计信息
func GetStats() map[string]interface{} {
	workflowCount, activityCount := core.GetGlobal().Count() // 获取全局注册的工作流和活动数量

	return map[string]interface{}{
		"initialized":     initialized,           // SDK 是否已初始化
		"workflow_count":  workflowCount,         // 注册的工作流数量
		"activity_count":  activityCount,         // 注册的活动数量
		"worker_count":    core.GetWorkerCount(), // 工作者数量
		"workers_running": core.IsRunning(),      // 工作者是否正在运行
	}
}

// validateConfig 验证配置的有效性
func validateConfig(cfg *config.Config) error {
	if cfg.HostPort == "" {
		return fmt.Errorf("host_port is required") // 检查 HostPort 是否为空
	}
	if cfg.Namespace == "" {
		return fmt.Errorf("namespace is required") // 检查 Namespace 是否为空
	}
	if len(cfg.Workers) == 0 {
		return fmt.Errorf("at least one worker configuration is required") // 检查至少需要一个工作者配置
	}
	for i, workerCfg := range cfg.Workers {
		if workerCfg.TaskQueue == "" {
			return fmt.Errorf("worker[%d] task_queue is required", i) // 检查每个工作者的 TaskQueue 是否为空
		}
	}
	return nil
}
