package core

import (
	"fmt"
	"sync"
)

// Registry 管理工作流和活动的注册
type Registry struct {
	mu         sync.RWMutex
	workflows  map[string]interface{}
	activities map[string]interface{}
}

// Module 表示一个可注册的模块
type Module interface {
	RegisterWorkflows(r *Registry)
	RegisterActivities(r *Registry)
	Name() string
}

// WorkerRegistry 用于 Worker 注册的接口
type WorkerRegistry interface {
	RegisterWorkflow(workflow interface{})
	RegisterActivity(activity interface{})
}

var globalRegistry = NewRegistry()

// NewRegistry 创建一个新的注册表
func NewRegistry() *Registry {
	return &Registry{
		workflows:  make(map[string]interface{}),
		activities: make(map[string]interface{}),
	}
}

// RegisterWorkflow 全局注册一个工作流
func RegisterWorkflow(name string, workflow interface{}) {
	globalRegistry.RegisterWorkflow(name, workflow)
}

// RegisterActivity 全局注册一个活动
func RegisterActivity(name string, activity interface{}) {
	globalRegistry.RegisterActivity(name, activity)
}

// RegisterModule 注册一个完整的模块
func RegisterModule(module Module) {
	fmt.Printf("Registering module: %s\n", module.Name())
	module.RegisterWorkflows(globalRegistry)
	module.RegisterActivities(globalRegistry)
}

// RegisterWorkflow 注册一个工作流
func (r *Registry) RegisterWorkflow(name string, workflow interface{}) {
	r.mu.Lock()
	defer r.mu.Unlock()
	fmt.Printf("Registering workflow: %s\n", name)
	r.workflows[name] = workflow
}

// RegisterActivity 注册一个活动
func (r *Registry) RegisterActivity(name string, activity interface{}) {
	r.mu.Lock()
	defer r.mu.Unlock()
	fmt.Printf("Registering activity: %s\n", name)
	r.activities[name] = activity
}

// ApplyTo 将所有注册项应用到一个 Worker 上
func (r *Registry) ApplyTo(w WorkerRegistry) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for name, workflow := range r.workflows {
		fmt.Printf("Applying workflow to worker: %s\n", name)
		w.RegisterWorkflow(workflow)
	}

	for name, activity := range r.activities {
		fmt.Printf("Applying activity to worker: %s\n", name)
		w.RegisterActivity(activity)
	}
}

// GetWorkflows 返回所有已注册的工作流
func (r *Registry) GetWorkflows() map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]interface{})
	for k, v := range r.workflows {
		result[k] = v
	}
	return result
}

// GetActivities 返回所有已注册的活动
func (r *Registry) GetActivities() map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]interface{})
	for k, v := range r.activities {
		result[k] = v
	}
	return result
}

// GetGlobal 返回全局注册表实例
func GetGlobal() *Registry {
	return globalRegistry
}

// Count 返回已注册的工作流和活动数量
func (r *Registry) Count() (workflows int, activities int) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.workflows), len(r.activities)
}
