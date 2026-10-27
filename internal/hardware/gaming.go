package hardware

// Gaming is the set of abilities that the layers above the hardware use.
//
// The interface keeps the program testable: a test can hand the layers a
// fake that stands for the machine, so that the behaviour of the program
// can be checked without the firmware and without the rights of an
// administrator.
type Gaming interface {
	// Instance returns the object path of the gaming functions.
	Instance() string

	// Sensors reads the live temperatures and fan speeds.
	Sensors() (Sensors, error)

	// PowerMode returns the performance profile.
	PowerMode() (PowerMode, error)
	// SetPowerMode applies a performance profile.
	SetPowerMode(mode PowerMode) error

	// GPUMode returns the graphics switch of the machine.
	GPUMode() (GPUMode, error)
	// SetGPUMode selects the graphics switch of the machine.
	SetGPUMode(mode GPUMode) error

	// LightingBrightness reads the brightness of the keyboard.
	LightingBrightness() (uint64, error)
	// Lighting reads the keyboard lighting.
	Lighting() (RGBConfig, error)
	// SetLighting applies a keyboard lighting configuration.
	SetLighting(config RGBConfig) error
	// ResetLighting restores the default keyboard lighting.
	ResetLighting() error

	// Backlight reads the keyboard backlight.
	Backlight() (BacklightState, error)
	// SetBacklight sets the brightness of the keyboard backlight.
	SetBacklight(brightness int, timeout bool) error

	// Fans reads the fan configuration.
	Fans() (FanState, error)
	// SetFans applies a fan configuration.
	SetFans(config FanConfig) error

	// CoolBoostEnabled reports whether the boost fan table is selected.
	CoolBoostEnabled() (bool, error)
	// CoolBoost switches the boost fan table.
	CoolBoost(enabled bool) error

	// WinKeyLocked reports whether the Windows key is locked.
	WinKeyLocked() (bool, error)
	// SetWinKeyLocked locks or unlocks the Windows key.
	SetWinKeyLocked(locked bool) error

	// LCDOverdrive reports whether the display overdrive is active.
	LCDOverdrive() (bool, error)
	// SetLCDOverdrive enables or disables the display overdrive.
	SetLCDOverdrive(enabled bool) error

	// BootSound reports whether the sound at the start of the machine is on.
	BootSound() (bool, error)
	// SetBootSound switches the sound at the start of the machine on or off.
	SetBootSound(enabled bool) error

	// DiscreteGPU reports the state of the discrete GPU switch.
	DiscreteGPU() (bool, error)

	// OverclockCapability reports which overclocking features the machine
	// supports.
	OverclockCapability() (gpu bool, cpu bool, err error)

	// Close releases the connection to the machine.
	Close()
}

// The backend is the implementation that talks to the firmware.
var _ Gaming = (*Backend)(nil)
