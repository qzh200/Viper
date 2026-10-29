// Package domain 定义游戏笔记本硬件控制与状态采集的核心领域接口与数据模型。
package domain

// PowerModeReader 定义电源模式读取接口。
type PowerModeReader interface {
	PowerMode() (PowerMode, error)
	SetPowerMode(mode PowerMode) error
}

// GraphicsModeReader 定义显卡工作模式与超频能力的读取与切换接口。
type GraphicsModeReader interface {
	GPUMode() (GPUMode, error)
	SetGPUMode(mode GPUMode) error
	DiscreteGPU() (bool, error)
	OverclockCapability() (gpu bool, cpu bool, err error)
}

// LightingReader 定义键盘背光与亮度读取接口。
type LightingReader interface {
	LightingBrightness() (uint64, error)
	Backlight() (BacklightState, error)
}

// LightingConfig 定义 RGB 灯效配置查询接口。
type LightingConfig interface {
	Lighting() (RGBConfig, error)
}

// LightingWriter 定义 RGB 灯效及背光参数设置接口。
type LightingWriter interface {
	SetLighting(config RGBConfig) error
	SetBacklight(brightness int, timeout bool) error
}

// LightingReset 定义灯效恢复默认接口。
type LightingReset interface {
	ResetLighting() error
}

// FansReader 定义风扇状态读取接口。
type FansReader interface {
	Fans() (FanState, error)
}

// FansWriter 定义风扇配置写入接口。
type FansWriter interface {
	SetFans(config FanConfig) error
}

// CoolBoostReader 定义 CoolBoost 强冷开关接口。
type CoolBoostReader interface {
	CoolBoostEnabled() (bool, error)
	CoolBoost(enabled bool) error
}

// SwitchesReader 定义外设与系统开关（Win键锁定、LCD响应时间超速、开机声音）的读写接口。
type SwitchesReader interface {
	WinKeyLocked() (bool, error)
	SetWinKeyLocked(locked bool) error
	LCDOverdrive() (bool, error)
	SetLCDOverdrive(enabled bool) error
	BootSound() (bool, error)
	SetBootSound(enabled bool) error
}

// ReadingsReader 定义传感器基础读数与生命周期接口。
type ReadingsReader interface {
	Instance() string
	Sensors() (Sensors, error)
	Close()
}

// Gaming 组合了笔记本硬件控制的所有读写能力。
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
