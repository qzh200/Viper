package hardware

import (
	"fmt"
)

// The values of the line status of the power supply.
const (
	// PowerOffline means that the machine runs on the battery.
	ACLineOffline = 0
	// PowerOnline means that the machine runs on the power supply.
	ACLineOnline = 1
	// PowerBackup means that the machine runs on a backup supply.
	ACLineBackup = 2
	// PowerUnknown means that the state cannot be told.
	ACLineUnknown = 255
)

// The bits of the battery flag.
const (
	// BatteryHigh means that the battery is nearly full.
	BatteryHigh = 1
	// BatteryLow means that the battery is running low.
	BatteryLow = 2
	// BatteryCritical means that the battery is almost empty.
	BatteryCritical = 4
	// BatteryCharging means that the battery is charged.
	BatteryCharging = 8
	// BatteryAbsent means that the machine has no battery.
	BatteryAbsent = 128
)

// PowerStatus describes the power supply of the machine.
type PowerStatus struct {
	// OnAC reports whether the machine runs on the power supply.
	OnAC bool
	// OnBattery reports whether the machine runs on the battery.
	OnBattery bool
	// BatteryPercent is the charge of the battery in percent, or 255 when
	// the machine has no battery.
	BatteryPercent int
	// Charging reports whether the battery is being charged.
	Charging bool
	// BatteryLow reports whether the battery is running low.
	BatteryLow bool
	// BatteryCritical reports whether the battery is almost empty.
	BatteryCritical bool
}

// String renders the power status for display.
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
