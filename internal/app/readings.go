package app

import (
	"viper/internal/domain"
)

// Telemetry 汇集原厂监控数据指标。
type Telemetry struct {
	Monitor domain.MonitorData
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
func (a *App) Sensors() (domain.Sensors, error) {
	backend, err := a.hardware()
	if err != nil {
		return domain.Sensors{}, err
	}
	return backend.Sensors()
}

// Monitor 获取全套软硬件运行参数与监控指标。
func (a *App) Monitor() (domain.MonitorData, error) {
	backend, err := a.hardware()
	if err != nil {
		return domain.MonitorData{}, err
	}
	return backend.Monitor()
}

// Telemetry 采集包含整机遥测的完整数据包。
func (a *App) Telemetry() (Telemetry, error) {
	monitor, err := a.Monitor()
	if err != nil {
		return Telemetry{}, err
	}

	return Telemetry{Monitor: monitor}, nil
}
