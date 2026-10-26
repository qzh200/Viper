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
// The provider reports the behaviour in the third byte, multiplied by two
// hundred and fifty six, and uses the same numbering as the value that is
// written: one is automatic, two is maximum and three is a fixed duty
// cycle.
func decodeFanMode(bits uint64) FanMode {
	switch bits {
	case 1:
		return FanAuto
	case 2:
		return FanMax
	case 3:
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
	// PowerUnknown is a profile the firmware does not describe.
	PowerUnknown
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

// providerCode maps the mode onto the encoding the provider expects.
//
// The encoding is not a plain enumeration and it is not the one the MOF
// suggests. Measured on a Predator Helios Neo 16, the provider keeps five
// values and rejects everything else:
//
//	0 -> quiet
//	1 -> balanced
//	4 -> performance
//	5 -> turbo
//	6 -> a profile the firmware keeps but that behaves like quiet under
//	     load, so it is not offered to the user
//
// Writing two or three leaves the previous profile in place, which is why
// the values look like an ordinary enumeration to anyone who only tries
// the obvious ones.
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
	default:
		return 1
	}
}

// decodePowerMode maps the provider's value back onto a mode.
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
	default:
		return PowerUnknown
	}
}

// AvailablePowerModes lists the profiles this machine supports.
//
// Turbo needs a Predator, the other profiles are available everywhere.
func AvailablePowerModes(isPredator bool) []PowerMode {
	if isPredator {
		return []PowerMode{PowerQuiet, PowerBalanced, PowerPerformance, PowerTurbo}
	}
	return []PowerMode{PowerQuiet, PowerBalanced, PowerPerformance}
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
	case "turbo", "extreme":
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
//
// The effect, the speed and the direction are global, because the provider
// applies them to the whole keyboard at once. Only the colour is set per
// zone, which is what the official application shows as the difference
// between a global and a per zone setup.
type RGBConfig struct {
	// Effect selects the lighting pattern.
	Effect LightingEffect
	// Speed is the animation speed.
	Speed uint64
	// Direction is the animation direction, zero or one.
	Direction uint64
	// Brightness is the overall brightness, 0 to 100.
	Brightness uint64
	// Zones holds the colour of each of the four lighting zones.
	Zones [4]RGBColor
	// PerZone reports whether the zones carry individual colours. When it
	// is false every zone uses the colour of the first one.
	PerZone bool
	// ColourReadable reports whether the firmware reports the zone
	// colours. It does not, so the caller keeps its own record.
	ColourReadable bool
	// EffectReadable reports whether the firmware reports the lighting
	// effect. It does not, so the caller keeps its own record.
	EffectReadable bool
}

// LightingEffect is one of the keyboard lighting patterns.
//
// The numbering is the one the Acer lighting service uses in its profile
// file, where every pattern carries a value field:
//
//	0 static      1 breathing   2 neon     3 wave
//	4 shifting    5 zoom        6 meteor   7 twinkling
//
// The order is the order in which the official application lists them.
type LightingEffect uint64

// The lighting patterns of the official application, in the order it
// lists them.
const (
	// EffectStatic keeps a constant colour.
	EffectStatic LightingEffect = 0
	// EffectBreathing fades the colour in and out.
	EffectBreathing LightingEffect = 1
	// EffectNeon lights the zones one after the other.
	EffectNeon LightingEffect = 2
	// EffectWave moves a highlight across the zones.
	EffectWave LightingEffect = 3
	// EffectSwitch alternates between two colours.
	EffectSwitch LightingEffect = 4
	// EffectScale grows and shrinks the highlight.
	EffectScale LightingEffect = 5
	// EffectMeteor runs a shower of lights.
	EffectMeteor LightingEffect = 6
	// EffectFlash blinks the keyboard.
	EffectFlash LightingEffect = 7
)

// String renders the effect for display.
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

// ParseLightingEffect maps a name onto a lighting effect.
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

// MiscSetting identifies one of the provider's miscellaneous settings.
//
// The provider keeps a small table of switches. A setting is addressed by
// its identifier and carries its value in the upper byte of the same word.
type MiscSetting uint32

// The miscellaneous settings that are known.
//
// Only the settings below the tenth identifier are writable, everything
// else answers with an undefined error.
const (
	// MiscBluelightShield selects the blue light shield.
	MiscBluelightShield MiscSetting = 0x01
	// MiscGPUMode selects the graphics switch of the machine.
	MiscGPUMode MiscSetting = 0x02
	// MiscUSBCharge holds the charging limit in percent.
	MiscUSBCharge MiscSetting = 0x06
	// MiscUSBChargeSwitch enables the charging over USB.
	MiscUSBChargeSwitch MiscSetting = 0x08
	// MiscDiscreteGPU reports the discrete GPU switch state.
	MiscDiscreteGPU MiscSetting = 0x09
	// MiscCPUOverclock reports the CPU overclock capability.
	MiscCPUOverclock MiscSetting = 0x0A
	// MiscOperationMode selects the performance profile.
	MiscOperationMode MiscSetting = 0x0B
)

// GPUMode is the way the machine drives the graphics chip.
//
// The official application offers three modes, the capability mask in the
// registry of the original service says which of them the machine has. A
// mask of seven offers all three, a mask of three only the two that keep
// the hybrid graphics switched on.
type GPUMode uint64

// The graphics modes the provider understands.
const (
	// GPUModeAuto lets the system pick the mode on its own.
	GPUModeAuto GPUMode = 1
	// GPUModeOptimus keeps the integrated graphics in charge.
	GPUModeOptimus GPUMode = 2
	// GPUModeDiscrete hands the display over to the dedicated graphics.
	GPUModeDiscrete GPUMode = 3
)

// String renders the mode for display.
func (m GPUMode) String() string {
	switch m {
	case GPUModeAuto:
		return "auto"
	case GPUModeOptimus:
		return "optimus"
	case GPUModeDiscrete:
		return "discrete"
	default:
		return "unknown"
	}
}

// ParseGPUMode maps a command line word onto a graphics mode.
func ParseGPUMode(name string) (GPUMode, bool) {
	switch name {
	case "auto", "automatic":
		return GPUModeAuto, true
	case "optimus", "hybrid":
		return GPUModeOptimus, true
	case "discrete", "dGPU", "nvidia":
		return GPUModeDiscrete, true
	default:
		return GPUModeOptimus, false
	}
}
