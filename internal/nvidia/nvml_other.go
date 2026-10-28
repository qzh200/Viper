//go:build !windows

package nvidia

import "errors"

var errNoDriver = errors.New("nvidia: 这个系统没有图形卡的驱动库")

// Telemetry 记录单个 NVIDIA GPU 的硬件遥测指标。
type Telemetry struct {
	Name                string
	UsagePercent        int
	MemoryUsagePercent  int
	GraphicsClockMHz    int
	MemoryClockMHz      int
	MaxGraphicsClockMHz int
	MemoryUsedBytes     uint64
	MemoryTotalBytes    uint64
	PState              int
	Idle                bool
}

// IsIdle 判断显卡是否处于空闲状态。
func (t Telemetry) IsIdle() bool {
	return t.UsagePercent == 0
}

// Available 在非 Windows 平台始终返回 false。
func Available() bool {
	return false
}

// Close 在非 Windows 平台为空操作。
func Close() {}

// Read 在非 Windows 平台返回不支持错误。
func Read() ([]Telemetry, error) {
	return nil, errNoDriver
}

// DriverVersion 在非 Windows 平台返回不支持错误。
func DriverVersion() (string, error) {
	return "", errNoDriver
}
