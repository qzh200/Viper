//go:build !windows

package hardware

import (
	"fmt"
)

// StickyKeys reports whether the sticky keys are switched on.
//
// 粘滞键是 Windows 的辅助功能设置，别的系统上没有这个设置。
func StickyKeys() (bool, error) {
	return false, fmt.Errorf("hardware: 粘滞键只有 Windows 支持")
}

// SetStickyKeys switches the sticky keys on or off.
//
// 粘滞键是 Windows 的辅助功能设置，别的系统上没有这个设置。
func SetStickyKeys(enabled bool) error {
	return fmt.Errorf("hardware: 粘滞键只有 Windows 支持")
}
