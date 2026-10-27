//go:build windows

package hardware

import (
	"fmt"
	"syscall"
	"unsafe"
)

// kernel32 holds the calls of the kernel library.
var (
	kernel32Library          = syscall.NewLazyDLL("kernel32.dll")
	procGetSystemPowerStatus = kernel32Library.NewProc("GetSystemPowerStatus")
)

// systemPowerStatus is the structure the system fills in.
type systemPowerStatus struct {
	ACLineStatus        byte
	BatteryFlag         byte
	BatteryLifePercent  byte
	Reserved1           byte
	BatteryLifeTime     uint32
	BatteryFullLifeTime uint32
}

// SystemPowerStatus reads the power supply of the machine.
//
// The information comes from the kernel of Windows, so it works without the
// services of the original software and without administrative rights.
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
		// The machine runs on a power supply without a battery.
		status.BatteryPercent = 0
	}
	return status, nil
}
