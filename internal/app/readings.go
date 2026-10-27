// 机器的读数：传感器、电源、显卡遥测、身份。
package app

import (
	"viper/internal/hardware"
	"viper/internal/nvidia"
)

// Instance returns the object path of the gaming functions.
func (a *App) Instance() (string, error) {
	backend, err := a.hardware()
	if err != nil {
		return "", err
	}
	return backend.Instance(), nil
}

// Sensors reads the live temperatures and fan speeds.
func (a *App) Sensors() (hardware.Sensors, error) {
	backend, err := a.hardware()
	if err != nil {
		return hardware.Sensors{}, err
	}
	return backend.Sensors()
}

// PowerStatus reads the power supply of the machine.
//
// The information comes from the kernel of Windows, so it needs neither the
// gaming provider nor the services of the original software.
func (a *App) PowerStatus() (hardware.PowerStatus, error) {
	return hardware.SystemPowerStatus()
}

// GPUTelemetry reads the state of the graphics card.
//
// The numbers come from the driver of the card. A machine without a card
// that offers them answers with an empty list.
func (a *App) GPUTelemetry() ([]nvidia.Telemetry, error) {
	cards, err := nvidia.Read()
	if err != nil {
		// A machine without the driver of the vendor simply has no
		// telemetry, that is not an error.
		if nvidia.Available() {
			return nil, err
		}
		return nil, nil
	}
	return cards, nil
}

// Identity returns the machine information from the firmware.
func (a *App) Identity() (hardware.Identity, error) {
	return hardware.SystemInfo()
}
