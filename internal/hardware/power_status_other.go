//go:build !windows

package hardware

import (
	"fmt"
)

// SystemPowerStatus 在非 Windows 平台上的存根实现，返回不支持错误。
func SystemPowerStatus() (PowerStatus, error) {
	return PowerStatus{}, fmt.Errorf("hardware: 电源状态只有 Windows 支持")
}
