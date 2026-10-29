package hardware

// FanMode 表示风扇运转模式。
type FanMode int

const (
	// FanAuto 表示风扇自动转速模式。
	FanAuto FanMode = iota
	// FanMax 表示风扇全速运转模式。
	FanMax
	// FanCustom 表示用户自定义转速模式。
	FanCustom
)

// String 返回风扇模式的文本名称。
func (m FanMode) String() string {
	switch m {
	case FanAuto:
		return "auto"
	case FanMax:
		return "max"
	case FanCustom:
		return "custom"
	default:
		return "unknown"
	}
}

// PowerMode 表示系统电源/性能配置模式。
type PowerMode int

const (
	// PowerQuiet 表示静音/低功耗模式。
	PowerQuiet PowerMode = iota
	// PowerBalanced 表示平衡模式。
	PowerBalanced
	// PowerPerformance 表示高性能模式。
	PowerPerformance
	// PowerTurbo 表示极速/狂暴模式。
	PowerTurbo
	// PowerUnknown 表示未识别模式。
	PowerUnknown
	// PowerEco 表示节能模式。
	PowerEco PowerMode = 5
)

// String 返回电源模式的标识符。
func (m PowerMode) String() string {
	switch m {
	case PowerQuiet:
		return "quiet"
	case PowerBalanced:
		return "balanced"
	case PowerPerformance:
		return "performance"
	case PowerTurbo:
		return "turbo"
	case PowerEco:
		return "eco"
	default:
		return "unknown"
	}
}

func decodePowerMode(value uint64) PowerMode {
	switch value {
	case 0:
		return PowerQuiet
	case 1:
		return PowerBalanced
	case 4:
		return PowerPerformance
	case 5:
		return PowerTurbo
	case 6:
		return PowerEco
	default:
		return PowerUnknown
	}
}

// PowerModeCode 返回对应底层提供方的数值编码。
func PowerModeCode(mode PowerMode) uint64 {
	return mode.providerCode()
}

func (m PowerMode) providerCode() uint64 {
	switch m {
	case PowerQuiet:
		return 0
	case PowerBalanced:
		return 1
	case PowerPerformance:
		return 4
	case PowerTurbo:
		return 5
	case PowerEco:
		return 6
	default:
		return 1
	}
}

// DecodePowerMode 将底层数值编码转换为对应的 PowerMode。
func DecodePowerMode(value uint64) PowerMode {
	return decodePowerMode(value)
}

// AvailablePowerModes 返回机型支持的可用电源模式列表。
func AvailablePowerModes() []PowerMode {
	return []PowerMode{PowerQuiet, PowerBalanced, PowerPerformance, PowerTurbo, PowerEco}
}

// ParsePowerMode 根据名称解析电源模式。
func ParsePowerMode(name string) (PowerMode, bool) {
	switch name {
	case "quiet":
		return PowerQuiet, true
	case "balanced", "default":
		return PowerBalanced, true
	case "performance", "perf":
		return PowerPerformance, true
	case "turbo", "extreme":
		return PowerTurbo, true
	case "eco", "power_saver", "saver", "energy_saver":
		return PowerEco, true
	default:
		return PowerBalanced, false
	}
}

// RGBColor 表示 24 位 RGB 颜色分量。
type RGBColor struct {
	R uint8
	G uint8
	B uint8
}

// RGBConfig 描述背光灯效的完整配置参数。
type RGBConfig struct {
	// Effect 指定灯效模式。
	Effect LightingEffect
	// Speed 指定动态灯效速度。
	Speed uint64
	// Direction 指定灯效流动方向（如 0 或 1）。
	Direction uint64
	// Brightness 指定亮度百分比或等级。
	Brightness uint64
	// Zones 包含各分区颜色（支持 4 分区）。
	Zones [4]RGBColor
	// PerZone 标识是否采用多分区独立颜色。
	PerZone bool
	// ColourReadable 标识颜色配置是否可读。
	ColourReadable bool
	// EffectReadable 标识灯效模式是否可读。
	EffectReadable bool
}

// LightingEffect 表示背光动画效果类型。
type LightingEffect uint64

const (
	// EffectStatic 静态常亮。
	EffectStatic LightingEffect = 0
	// EffectBreathing 呼吸渐变。
	EffectBreathing LightingEffect = 1
	// EffectNeon 霓虹变色。
	EffectNeon LightingEffect = 2
	// EffectWave 波浪滚动。
	EffectWave LightingEffect = 3
	// EffectSwitch 交叉切换。
	EffectSwitch LightingEffect = 4
	// EffectScale 缩放脉冲。
	EffectScale LightingEffect = 5
	// EffectMeteor 流星划过。
	EffectMeteor LightingEffect = 6
	// EffectFlash 闪烁模式。
	EffectFlash LightingEffect = 7
)

// String 返回灯效的文本表示。
func (e LightingEffect) String() string {
	switch e {
	case EffectStatic:
		return "static"
	case EffectBreathing:
		return "breathing"
	case EffectNeon:
		return "neon"
	case EffectWave:
		return "wave"
	case EffectSwitch:
		return "switch"
	case EffectScale:
		return "scale"
	case EffectMeteor:
		return "meteor"
	case EffectFlash:
		return "flash"
	default:
		return "unknown"
	}
}

// ParseLightingEffect 根据字符串解析灯效类型。
func ParseLightingEffect(name string) (LightingEffect, bool) {
	switch name {
	case "static":
		return EffectStatic, true
	case "breathing", "breathe":
		return EffectBreathing, true
	case "neon":
		return EffectNeon, true
	case "wave":
		return EffectWave, true
	case "switch", "alternate":
		return EffectSwitch, true
	case "scale", "zoom":
		return EffectScale, true
	case "meteor", "meteorshower":
		return EffectMeteor, true
	case "flash", "strobe":
		return EffectFlash, true
	default:
		return EffectStatic, false
	}
}

// GPUMode 表示显卡切换与工作模式。
type GPUMode uint64

const (
	// GPUModeHybrid 混合模式（核显+独显 Dynamic/Optimus）。
	GPUModeHybrid GPUMode = 1
	// GPUModeDiscrete 纯独显直连模式。
	GPUModeDiscrete GPUMode = 2
	// GPUModeAuto 自动显卡切换模式。
	GPUModeAuto GPUMode = 3
)

// String 返回显卡模式的名称。
func (m GPUMode) String() string {
	switch m {
	case GPUModeHybrid:
		return "hybrid"
	case GPUModeDiscrete:
		return "discrete"
	case GPUModeAuto:
		return "auto"
	default:
		return "unknown"
	}
}

// ParseGPUMode 将模式名称解析为 GPUMode。
func ParseGPUMode(name string) (GPUMode, bool) {
	switch name {
	case "hybrid", "optimus", "mshybrid":
		return GPUModeHybrid, true
	case "discrete", "dgpu", "nvidia", "only":
		return GPUModeDiscrete, true
	case "auto", "automatic", "dds":
		return GPUModeAuto, true
	default:
		return GPUModeHybrid, false
	}
}

func clampUint(value, low, high uint64) uint64 {
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
}

// LightingZones 定义支持的背光分区数量。
const LightingZones = 4

// Sensors 包含基本的温度与转速传感器数据。
type Sensors struct {
	CPUTempC    int
	CPUFanRPM   int
	GPUTempC    int
	GPUFanRPM   int
	SystemTempC int
}

// FanState 记录风扇当前的模式与运行状态。
type FanState struct {
	CPUMode         FanMode
	GPUMode         FanMode
	GPUModeReadable bool
	CPUPercent      int
	GPUPercent      int
	CoolBoost       bool
}

// FanConfig 定义待应用的风扇目标配置。
type FanConfig struct {
	CPUMode    FanMode
	GPUMode    FanMode
	CPUPercent int
	GPUPercent int
	CoolBoost  bool
}

// BacklightState 记录键盘背光亮度与自动超时关闭状态。
type BacklightState struct {
	Brightness int
	Timeout    bool
}
