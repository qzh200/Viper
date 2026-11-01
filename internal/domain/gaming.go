package domain

type PowerModeReader interface {
	PowerMode() (PowerMode, error)
	SetPowerMode(mode PowerMode) error
}

type GraphicsModeReader interface {
	GPUMode() (GPUMode, error)
	SetGPUMode(mode GPUMode) error
	DiscreteGPU() (bool, error)
	OverclockCapability() (gpu bool, cpu bool, err error)
}

type LightingReader interface {
	LightingBrightness() (uint64, error)
	Backlight() (BacklightState, error)
}

type LightingConfig interface {
	Lighting() (RGBConfig, error)
}

type LightingWriter interface {
	SetLighting(config RGBConfig) error
	SetBacklight(brightness int, timeout bool) error
}

type LightingReset interface {
	ResetLighting() error
}

type FansReader interface {
	Fans() (FanState, error)
}

type FansWriter interface {
	SetFans(config FanConfig) error
}

type CoolBoostReader interface {
	CoolBoostEnabled() (bool, error)
	CoolBoost(enabled bool) error
}

type SwitchesReader interface {
	WinKeyLocked() (bool, error)
	SetWinKeyLocked(locked bool) error
	LCDOverdrive() (bool, error)
	SetLCDOverdrive(enabled bool) error
	BootSound() (bool, error)
	SetBootSound(enabled bool) error
}

type ReadingsReader interface {
	Instance() string
	Sensors() (Sensors, error)
	Close()
}

type Gaming interface {
	ReadingsReader
	PowerModeReader
	GraphicsModeReader
	LightingReader
	LightingConfig
	LightingWriter
	LightingReset
	FansReader
	FansWriter
	CoolBoostReader
	SwitchesReader

	Monitor() (MonitorData, error)
}
