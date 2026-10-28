package app

import (
	"viper/internal/hardware"
	"viper/internal/nvidia"
)

// Telemetry 汇集原厂监控数据与 NVIDIA 显卡专属遥测指标。
type Telemetry struct {
	Monitor hardware.MonitorData
	Cards   []nvidia.Telemetry
}

// Instance 返回当前底层连接的后端驱动实例名称。
func (a *App) Instance() (string, error) {
	backend, err := a.hardware()
	if err != nil {
		return "", err
	}
	return backend.Instance(), nil
}

// Sensors 读取基础传感器数据（CPU/GPU 温度与风扇转速）。
func (a *App) Sensors() (hardware.Sensors, error) {
	backend, err := a.hardware()
	if err != nil {
		return hardware.Sensors{}, err
	}
	return backend.Sensors()
}

// Monitor 获取全套软硬件运行参数与监控指标。
func (a *App) Monitor() (hardware.MonitorData, error) {
	backend, err := a.hardware()
	if err != nil {
		return hardware.MonitorData{}, err
	}
	return backend.Monitor()
}

// PowerStatus 查询当前适配器连接与电池充放电状态。
func (a *App) PowerStatus() (hardware.PowerStatus, error) {
	return hardware.SystemPowerStatus()
}

// Telemetry 采集包含整机遥测与 NVML 显卡详细指标的完整数据包。
func (a *App) Telemetry() (Telemetry, error) {
	monitor, err := a.Monitor()
	if err != nil {
		return Telemetry{}, err
	}

	cards, err := nvidia.Read()
	if err != nil {
		if nvidia.Available() {
			return Telemetry{}, err
		}
		cards = nil
	}

	return Telemetry{Monitor: monitor, Cards: cards}, nil
}

// Identity 读取 SMBIOS 固件信息以获取设备型号与厂商身份。
func (a *App) Identity() (hardware.Identity, error) {
	return hardware.SystemInfo()
}
