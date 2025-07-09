package core

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/og-game/glib/flowcore/config"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"google.golang.org/protobuf/types/known/durationpb"
)

// NamespaceManager 命名空间管理器
type NamespaceManager struct {
	client client.NamespaceClient
}

// NewNamespaceManager 创建命名空间管理器
func NewNamespaceManager(cfg *config.Config) (*NamespaceManager, error) {
	namespaceClient, err := createNamespaceClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create namespace client: %w", err)
	}

	return &NamespaceManager{
		client: namespaceClient,
	}, nil
}

// EnsureNamespaceExists 确保命名空间存在，不存在则创建
func (nm *NamespaceManager) EnsureNamespaceExists(ctx context.Context, cfg *config.Config) error {
	// 检查命名空间是否存在
	exists, err := nm.CheckNamespaceExists(ctx, cfg.Namespace)
	if err != nil {
		return fmt.Errorf("failed to check namespace existence: %w", err)
	}

	if exists {
		fmt.Printf("✅ Namespace '%s' already exists\n", cfg.Namespace)
		return nil
	}

	// 命名空间不存在，创建它
	fmt.Printf("🚀 Creating namespace '%s'...\n", cfg.Namespace)
	err = nm.CreateNamespace(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to create namespace: %w", err)
	}

	fmt.Printf("✅ Successfully created namespace '%s'\n", cfg.Namespace)
	return nil
}

// CheckNamespaceExists 检查命名空间是否存在
func (nm *NamespaceManager) CheckNamespaceExists(ctx context.Context, namespaceName string) (bool, error) {
	_, err := nm.client.Describe(ctx, namespaceName)
	if err == nil {
		return true, nil
	}

	// 检查是否是"命名空间不存在"的错误
	if IsNamespaceNotFoundError(err) {
		return false, nil
	}

	// 其他错误
	return false, err
}

// CreateNamespace 创建命名空间
func (nm *NamespaceManager) CreateNamespace(ctx context.Context, cfg *config.Config) error {
	// 解析保留期
	retentionPeriod := ParseRetentionPeriod(cfg.NamespaceConf.RetentionPeriod)

	// 设置描述
	description := cfg.NamespaceConf.Description
	if description == "" {
		description = fmt.Sprintf("Auto-created namespace for %s", cfg.Namespace)
	}

	// 创建命名空间请求
	request := &workflowservice.RegisterNamespaceRequest{
		Namespace:                        cfg.Namespace,
		Description:                      description,
		WorkflowExecutionRetentionPeriod: retentionPeriod,
		Data:                             make(map[string]string),
		IsGlobalNamespace:                false,
		HistoryArchivalState:             GetArchivalState(cfg.NamespaceConf.HistoryArchivalEnabled),
		VisibilityArchivalState:          GetArchivalState(cfg.NamespaceConf.VisibilityArchivalEnabled),
	}

	return nm.client.Register(ctx, request)
}

// Close 关闭命名空间管理器
func (nm *NamespaceManager) Close() {
	if nm.client != nil {
		nm.client.Close()
	}
}

// createNamespaceClient 创建命名空间客户端
func createNamespaceClient(cfg *config.Config) (client.NamespaceClient, error) {
	options := client.Options{
		HostPort: cfg.HostPort,
	}

	// Configure TLS if enabled
	if cfg.TLS.Enabled {
		tlsConfig, err := cfg.TLS.GetTLSConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to configure TLS: %w", err)
		}
		options.ConnectionOptions = client.ConnectionOptions{
			TLS: tlsConfig,
		}
	}

	return client.NewNamespaceClient(options)
}

// ParseRetentionPeriod 解析保留期
func ParseRetentionPeriod(retentionStr string) *durationpb.Duration {
	if retentionStr == "" {
		// 默认保留期：3天
		return durationpb.New(72 * time.Hour)
	}

	duration, err := time.ParseDuration(retentionStr)
	if err != nil {
		// 解析失败，使用默认值
		fmt.Printf("⚠️ Invalid retention period '%s', using default 72h\n", retentionStr)
		return durationpb.New(72 * time.Hour)
	}

	return durationpb.New(duration)
}

// GetArchivalState 获取归档状态
func GetArchivalState(enabled bool) enums.ArchivalState {
	if enabled {
		return enums.ARCHIVAL_STATE_ENABLED
	}
	return enums.ARCHIVAL_STATE_DISABLED
}

// IsNamespaceNotFoundError 判断是否为命名空间不存在错误
func IsNamespaceNotFoundError(err error) bool {
	if err == nil {
		return false
	}

	errMsg := strings.ToLower(err.Error())
	return strings.Contains(errMsg, "namespace") &&
		(strings.Contains(errMsg, "not found") ||
			strings.Contains(errMsg, "does not exist") ||
			strings.Contains(errMsg, "unknown namespace"))
}

// EnsureNamespaceExists 全局函数，用于在客户端创建时调用
func EnsureNamespaceExists(ctx context.Context, cfg *config.Config) error {
	nsManager, err := NewNamespaceManager(cfg)
	if err != nil {
		return fmt.Errorf("failed to create namespace manager: %w", err)
	}
	defer nsManager.Close()

	return nsManager.EnsureNamespaceExists(ctx, cfg)
}

// CreateMultipleNamespaces 批量创建命名空间
func CreateMultipleNamespaces(ctx context.Context, cfg *config.Config, namespaces []string) error {
	nsManager, err := NewNamespaceManager(cfg)
	if err != nil {
		return fmt.Errorf("failed to create namespace manager: %w", err)
	}
	defer nsManager.Close()

	for _, ns := range namespaces {
		// 为每个命名空间创建单独的配置
		nsCfg := *cfg
		nsCfg.Namespace = ns

		err = nsManager.EnsureNamespaceExists(ctx, &nsCfg)
		if err != nil {
			return fmt.Errorf("failed to ensure namespace '%s': %w", ns, err)
		}
	}

	return nil
}

// ValidateNamespace 验证命名空间配置
func ValidateNamespace(ctx context.Context, cfg *config.Config) error {
	nsManager, err := NewNamespaceManager(cfg)
	if err != nil {
		return fmt.Errorf("failed to create namespace manager: %w", err)
	}
	defer nsManager.Close()

	exists, err := nsManager.CheckNamespaceExists(ctx, cfg.Namespace)
	if err != nil {
		return fmt.Errorf("failed to check namespace: %w", err)
	}

	if !exists {
		return fmt.Errorf("namespace '%s' does not exist", cfg.Namespace)
	}

	fmt.Printf("✅ Namespace '%s' exists and is valid\n", cfg.Namespace)
	return nil
}
