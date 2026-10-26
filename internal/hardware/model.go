package hardware

// FanMode is the behaviour of a fan group.
type FanMode int

// The fan behaviours the ACPI provider understands.
const (
	// FanAuto lets the embedded controller follow its own curve.
	FanAuto FanMode = iota
	// FanMax drives the fan at full speed.
	FanMax
	// FanCustom drives the fan at a fixed duty cycle.
	FanCustom
)

// String renders the mode for display.
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

// providerCode maps the mode onto the bit encoding the EC expects.
func (m FanMode) providerCode() uint64 {
	switch m {
	case FanMax:
		return 2
	case FanCustom:
		return 3
	default:
		return 1
	}
}

// decodeFanMode maps the provider's behaviour bits back onto a mode.
//
// The provider writes one, two or three but answers with zero for auto, so
// the reading is one lower than the written value.
func decodeFanMode(bits uint64) FanMode {
	switch bits {
	case 1:
		return FanMax
	case 2:
		return FanCustom
	default:
		return FanAuto
	}
}

// PowerMode is the performance profile of the machine.
type PowerMode int

// The performance profiles the provider exposes.
const (
	// PowerQuiet favours silence over performance.
	PowerQuiet PowerMode = iota
	// PowerBalanced is the default profile.
	PowerBalanced
	// PowerPerformance favours performance.
	PowerPerformance
	// PowerTurbo is the maximum profile, available on Predator models only.
	PowerTurbo
)

// String renders the mode for display.
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
	default:
		return "unknown"
	}
}

// ParsePowerMode maps a command line word onto a power mode.
func ParsePowerMode(name string) (PowerMode, bool) {
	switch name {
	case "quiet":
		return PowerQuiet, true
	case "balanced", "default":
		return PowerBalanced, true
	case "performance", "perf":
		return PowerPerformance, true
	case "turbo":
		return PowerTurbo, true
	default:
		return PowerBalanced, false
	}
}

// RGBColor is a colour of one keyboard zone.
type RGBColor struct {
	R uint8
	G uint8
	B uint8
}

// RGBConfig is the full keyboard lighting configuration.
type RGBConfig struct {
	// Effect selects the lighting pattern.
	Effect uint64
	// Speed is the animation speed, 1 to 5.
	Speed uint64
	// Direction is the animation direction.
	Direction uint64
	// Brightness is the overall brightness, 0 to 100.
	Brightness uint64
	// Zones holds the colour of each of the four lighting zones.
	Zones [4]RGBColor
}

// LED effect identifiers used by the provider.
const (
	// EffectStatic keeps a constant colour.
	EffectStatic uint64 = 0
	// EffectBreathing fades the colour in and out.
	EffectBreathing uint64 = 1
	// EffectWave moves a highlight across the zones.
	EffectWave uint64 = 2
	// EffectLightning flashes the zones.
	EffectLightning uint64 = 3
	// EffectSpectrum cycles through colours.
	EffectSpectrum uint64 = 4
)

// MiscSetting identifies one of the provider's miscellaneous settings.
type MiscSetting uint32

// The miscellaneous settings that are known.
const (
	// MiscOperationMode selects the performance profile.
	MiscOperationMode MiscSetting = 0x0B
	// MiscDiscreteGPU reports the discrete GPU switch state.
	MiscDiscreteGPU MiscSetting = 0x09
	// MiscCPUOverclock reports the CPU overclock capability.
	MiscCPUOverclock MiscSetting = 0x0A
)
