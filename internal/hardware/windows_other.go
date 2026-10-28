//go:build !windows

package hardware

import (
	"fmt"
)

// StickyKeys 在非 Windows 平台上的存根实现。
func StickyKeys() (bool, error) {
	return false, fmt.Errorf("hardware: 粘滞键只有 Windows 支持")
}

// SetStickyKeys 在非 Windows 平台上的存根实现。
func SetStickyKeys(enabled bool) error {
	return fmt.Errorf("hardware: 粘滞键只有 Windows 支持")
}
