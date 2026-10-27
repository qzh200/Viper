// 机器的读数：传感器、电源、显卡遥测、身份。
package app

import (
	"viper/internal/hardware"
	"viper/internal/nvidia"
)

// Telemetry is the state of the machine.
//
// 读数以**原厂服务**为主：它的监视数据里有处理器的使用率与频率、每核频率、
// 显卡的使用率与频率、内存的占用与频率、网络的上下行。
//
// 显卡驱动只**补充**它没报的那几样：显卡的名字、显存与显存频率。驱动报的
// 使用率与频率不用——原厂服务报的那些就是机器自己的数。
type Telemetry struct {
	// Monitor is what the original service reports about the machine.
	Monitor hardware.MonitorData
	// Cards are what the driver of the graphics cards reports: the name of
	// the card, the video memory and the memory clock. The service does not
	// report those.
	Cards []nvidia.Telemetry
}

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

// Monitor reads everything the machine reports about itself.
//
// 读数来自原厂服务的监视数据。
func (a *App) Monitor() (hardware.MonitorData, error) {
	backend, err := a.hardware()
	if err != nil {
		return hardware.MonitorData{}, err
	}
	return backend.Monitor()
}

// PowerStatus reads the power supply of the machine.
//
// The information comes from the kernel of Windows, so it needs neither the
// gaming provider nor the services of the original software.
func (a *App) PowerStatus() (hardware.PowerStatus, error) {
	return hardware.SystemPowerStatus()
}

// Telemetry reads the state of the machine: the readings of the original
// service first, the driver of the card filling in the rest.
//
// A machine without a card that offers them answers with an empty list of
// cards, the readings of the service are still there.
func (a *App) Telemetry() (Telemetry, error) {
	monitor, err := a.Monitor()
	if err != nil {
		return Telemetry{}, err
	}

	cards, err := nvidia.Read()
	if err != nil {
		// A machine without the driver of the vendor simply has no
		// telemetry from the card, that is not an error.
		if nvidia.Available() {
			return Telemetry{}, err
		}
		cards = nil
	}

	return Telemetry{Monitor: monitor, Cards: cards}, nil
}

// Identity returns the machine information from the machine's own tables.
func (a *App) Identity() (hardware.Identity, error) {
	return hardware.SystemInfo()
}
