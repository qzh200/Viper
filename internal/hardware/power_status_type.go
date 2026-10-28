package hardware

import (
	"fmt"
)

// AC 电源连接状态常量。
const (
	ACLineOffline = 0
	ACLineOnline  = 1
	ACLineBackup  = 2
	ACLineUnknown = 255
)

// 电池标志位常量（对应 Win32 SYSTEM_POWER_STATUS.BatteryFlag）。
const (
	BatteryHigh     = 1
	BatteryLow      = 2
	BatteryCritical = 4
	BatteryCharging = 8
	BatteryAbsent   = 128
)

// PowerStatus 记录当前系统电源与电池充放电状态。
type PowerStatus struct {
	// OnAC 标识当前是否连接交流适配器供电。
	OnAC bool

	// OnBattery 标识当前是否仅依赖电池供电。
	OnBattery bool

	// BatteryPercent 记录当前电池剩余百分比 (0-100)。
	BatteryPercent int

	// Charging 标识电池当前是否处于充电状态。
	Charging bool

	// BatteryLow 标识电池电量是否偏低。
	BatteryLow bool

	// BatteryCritical 标识电池电量是否进入告急状态。
	BatteryCritical bool
}

// String 返回电源状态的简要描述。
func (p PowerStatus) String() string {
	switch {
	case p.OnAC:
		if p.Charging {
			return "on the power supply, charging"
		}
		return "on the power supply"
	case p.OnBattery:
		return fmt.Sprintf("on the battery at %d percent", p.BatteryPercent)
	default:
		return "unknown"
	}
}
