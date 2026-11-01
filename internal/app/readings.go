package app

import (
	"viper/internal/domain"
)

type Telemetry struct {
	Monitor domain.MonitorData
}

func (a *App) Instance() (string, error) {
	backend, err := a.hardware()
	if err != nil {
		return "", err
	}
	return backend.Instance(), nil
}

func (a *App) Sensors() (domain.Sensors, error) {
	backend, err := a.hardware()
	if err != nil {
		return domain.Sensors{}, err
	}
	return backend.Sensors()
}

func (a *App) Monitor() (domain.MonitorData, error) {
	backend, err := a.hardware()
	if err != nil {
		return domain.MonitorData{}, err
	}
	return backend.Monitor()
}

func (a *App) Telemetry() (Telemetry, error) {
	monitor, err := a.Monitor()
	if err != nil {
		return Telemetry{}, err
	}

	return Telemetry{Monitor: monitor}, nil
}
