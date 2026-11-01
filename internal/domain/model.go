package domain

type FanMode int

const (
	FanAuto FanMode = iota
	FanMax
	FanCustom
)

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

type PowerMode int

const (
	PowerQuiet PowerMode = iota
	PowerBalanced
	PowerPerformance
	PowerTurbo
	PowerUnknown
	PowerEco PowerMode = 5
)

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

func DecodePowerMode(value uint64) PowerMode {
	return decodePowerMode(value)
}

func AvailablePowerModes() []PowerMode {
	return []PowerMode{PowerQuiet, PowerBalanced, PowerPerformance, PowerTurbo, PowerEco}
}

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

type RGBColor struct {
	R uint8
	G uint8
	B uint8
}

const (
	BrightnessMin uint64 = 1
	BrightnessMax uint64 = 5
)

const (
	SpeedMin uint64 = 1
	SpeedMax uint64 = 9
)

type RGBConfig struct {
	Effect         LightingEffect
	Speed          uint64
	Direction      uint64
	Brightness     uint64
	Zones          [4]RGBColor
	PerZone        bool
	ColourReadable bool
	EffectReadable bool
}

func (e LightingEffect) HasDirection() bool {
	return e == EffectWave || e == EffectSwitch
}

type LightingEffect uint64

const (
	EffectStatic    LightingEffect = 0
	EffectBreathing LightingEffect = 1
	EffectNeon      LightingEffect = 2
	EffectWave      LightingEffect = 3
	EffectSwitch    LightingEffect = 4
	EffectScale     LightingEffect = 5
	EffectMeteor    LightingEffect = 6
	EffectFlash     LightingEffect = 7
)

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

type GPUMode uint64

const (
	GPUModeHybrid   GPUMode = 1
	GPUModeDiscrete GPUMode = 2
	GPUModeAuto     GPUMode = 3
)

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

const LightingZones = 4

type Sensors struct {
	CPUTempC    int
	CPUFanRPM   int
	GPUTempC    int
	GPUFanRPM   int
	SystemTempC int
}

type FanState struct {
	CPUMode         FanMode
	GPUMode         FanMode
	GPUModeReadable bool
	CPUPercent      int
	GPUPercent      int
	CoolBoost       bool
}

type FanConfig struct {
	CPUMode    FanMode
	GPUMode    FanMode
	CPUPercent int
	GPUPercent int
	CoolBoost  bool
}

type BacklightState struct {
	Brightness int
	Timeout    bool
}
