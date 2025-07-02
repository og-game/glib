package snowx

import (
	"errors"
	"fmt"
	"github.com/sony/sonyflake"
	"net"
	"os"
	"strings"
	"time"
)

var flake *sonyflake.Sonyflake

func init() {
	startTime, _ := time.Parse("2006-01-02 15:04:05", "2025-07-01 00:00:00")
	flake = sonyflake.NewSonyflake(sonyflake.Settings{
		StartTime: startTime,
		MachineID: getMachineID,
	})
}

// GenId 生成一个唯一的雪花ID
func GenId() (id uint64, err error) {
	id, err = flake.NextID()
	return
}

// 获取机器 ID 基于 Docker 环境
func getMachineID() (uint16, error) {
	// 判断是否在 Docker 环境中运行
	if isRunningInDocker() {
		// 尝试通过容器 ID 生成机器 ID
		containerID, err := getContainerID()
		if err != nil {
			return 0, fmt.Errorf("failed to get container ID: %v", err)
		}
		return uint16(sum([]byte(containerID)) % 1024), nil
	}

	// 如果不在 Docker 环境中，继续使用 MAC 地址方式
	return getMachineIDFromMac()
}

// 判断是否在 Docker 容器中运行
func isRunningInDocker() bool {
	// 检查容器的特征文件
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}
	// 检查 cgroup 信息
	data, err := os.ReadFile("/proc/1/cgroup")
	if err != nil {
		return false
	}
	return strings.Contains(string(data), "docker")
}

// 获取容器 ID（适用于 Docker 环境）
func getContainerID() (string, error) {
	// 获取容器 ID
	// 一般情况下，可以通过读取 `/proc/self/cgroup` 获取容器 ID
	data, err := os.ReadFile("/proc/self/cgroup")
	if err != nil {
		return "", fmt.Errorf("failed to read /proc/self/cgroup: %v", err)
	}

	// 从文件内容中提取容器 ID
	for _, line := range strings.Split(string(data), "\n") {
		if strings.Contains(line, "docker") {
			parts := strings.Split(line, "/")
			if len(parts) > 2 {
				return parts[len(parts)-1], nil
			}
		}
	}
	return "", errors.New("container ID not found")
}

// 获取机器 ID 基于 MAC 地址（不在 Docker 环境下）
func getMachineIDFromMac() (uint16, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return 0, fmt.Errorf("failed to get network interfaces: %v", err)
	}

	// 查找第一个有效的网卡并获取其 MAC 地址
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.HardwareAddr == nil || len(iface.HardwareAddr) == 0 {
			continue
		}
		return uint16(sum(iface.HardwareAddr) % 1024), nil
	}

	return 0, errors.New("no valid network interface with MAC address found")
}

// 计算字节数组的和作为机器 ID
func sum(data []byte) int {
	total := 0
	for _, b := range data {
		total += int(b)
	}
	return total
}
