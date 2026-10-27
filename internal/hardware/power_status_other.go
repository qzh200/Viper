//go:build !windows

package hardware

import (
	"fmt"
)

// SystemPowerStatus reads the power supply of the machine.
//
// 电源状态来自 Windows 的内核，别的系统上没有这个来源，所以这里只报一个
// 不支持的错误。
func SystemPowerStatus() (PowerStatus, error) {
	return PowerStatus{}, fmt.Errorf("hardware: 电源状态只有 Windows 支持")
}
