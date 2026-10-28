//go:build windows

package hardware

import (
	"fmt"
	"syscall"
	"unsafe"
)

var (
	kernel32Library          = syscall.NewLazyDLL("kernel32.dll")
	procGetSystemPowerStatus = kernel32Library.NewProc("GetSystemPowerStatus")
)

// systemPowerStatus 对应 Win32 API 原生 SYSTEM_POWER_STATUS 内存结构。
type systemPowerStatus struct {
	ACLineStatus        byte
	BatteryFlag         byte
	BatteryLifePercent  byte
	Reserved1           byte
	BatteryLifeTime     uint32
	BatteryFullLifeTime uint32
}

// SystemPowerStatus 调用 Windows Win32 API 读取当前电源适配器与电池状态。
func SystemPowerStatus() (PowerStatus, error) {
	var raw systemPowerStatus

	result, _, callErr := procGetSystemPowerStatus.Call(
		uintptr(unsafe.Pointer(&raw)),
	)
	if result == 0 {
		return PowerStatus{}, fmt.Errorf("hardware: the power status is not available: %w", callErr)
	}

	status := PowerStatus{
		OnAC:            raw.ACLineStatus == ACLineOnline,
		OnBattery:       raw.ACLineStatus == ACLineOffline,
		BatteryPercent:  int(raw.BatteryLifePercent),
		Charging:        raw.BatteryFlag&BatteryCharging != 0,
		BatteryLow:      raw.BatteryFlag&BatteryLow != 0,
		BatteryCritical: raw.BatteryFlag&BatteryCritical != 0,
	}
	if raw.BatteryFlag&BatteryAbsent != 0 {
		status.BatteryPercent = 0
	}
	return status, nil
}
