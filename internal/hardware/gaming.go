package hardware

// The abilities of the machine are grouped by what they control, so that
// the layers above only depend on the parts they actually use.
//
// The interfaces keep the program testable: a test can hand the layers a
// stand for the machine, so that the behaviour of the program can be
// checked without the firmware and without the rights of an
// administrator.

// PowerMode is the performance profile of the machine.
type PowerModeReader interface {
	// PowerMode returns the performance profile.
	PowerMode() (PowerMode, error)
	// SetPowerMode applies a performance profile.
	SetPowerMode(mode PowerMode) error
}

// GraphicsMode is the way the machine drives the graphics chip.
type GraphicsModeReader interface {
	// GPUMode returns the graphics switch of the machine.
	GPUMode() (GPUMode, error)
	// SetGPUMode selects the graphics switch of the machine.
	SetGPUMode(mode GPUMode) error
	// DiscreteGPU reports the state of the discrete GPU switch.
	DiscreteGPU() (bool, error)
	// OverclockCapability reports which overclocking features the machine
	// supports.
	OverclockCapability() (gpu bool, cpu bool, err error)
}

// Lighting is the keyboard lighting and the backlight of the keyboard.
type LightingReader interface {
	// LightingBrightness reads the brightness of the keyboard.
	LightingBrightness() (uint64, error)
	// Backlight reads the keyboard backlight.
	Backlight() (BacklightState, error)
}

// LightingConfig is the lighting configuration of the keyboard.
//
// It is a part of the lighting on its own, because the layers above that
// only change the lighting never read the whole configuration back.
type LightingConfig interface {
	// Lighting reads the keyboard lighting.
	Lighting() (RGBConfig, error)
}

// LightingWriter is the writing of the keyboard lighting.
type LightingWriter interface {
	// SetLighting applies a keyboard lighting configuration.
	SetLighting(config RGBConfig) error
	// SetBacklight sets the brightness of the keyboard backlight.
	SetBacklight(brightness int, timeout bool) error
}

// LightingBuffer is the raw buffer of the keyboard backlight.
//
// It is a part of the lighting on its own, because the layers above that
// only report the lighting never read the raw buffer.
type LightingBuffer interface {
	// BacklightBuffer reads the raw buffer of the keyboard backlight.
	//
	// The buffer is sixteen bytes long, the level sits in byte two and in
	// byte five and byte eight holds the flags of the backlight.
	BacklightBuffer() ([]byte, error)
}

// LightingReset is the reset of the keyboard lighting.
//
// It is a part of the lighting on its own, because the layers above that
// only report and change the lighting never reset it.
type LightingReset interface {
	// ResetLighting restores the default keyboard lighting.
	ResetLighting() error
}

// Fans is the fan configuration of the machine.
//
// 读与写分成两个接口：原厂服务包了风扇的写入（它的风扇模式 1 就是全速），
// 固件那一侧只读不写，所以固件的提供者只满足读的那一个。
type FansReader interface {
	// Fans reads the fan configuration.
	Fans() (FanState, error)
}

// FansWriter is the writing of the fan configuration.
type FansWriter interface {
	// SetFans applies a fan configuration.
	SetFans(config FanConfig) error
}

// FanTableReader is the table of the fans.
//
// It is a part of the fan configuration on its own, because the layers
// above read the table without changing it.
type FanTableReader interface {
	// FanTable reads the selected fan table.
	FanTable() (uint64, error)
}

// CoolBoostReader is the boost of the fans.
//
// It is a part of the fan configuration on its own, because the layers
// above switch the boost without touching the duty cycles.
type CoolBoostReader interface {
	// CoolBoostEnabled reports whether the boost fan table is selected.
	CoolBoostEnabled() (bool, error)
	// CoolBoost switches the boost fan table.
	CoolBoost(enabled bool) error
}

// Switches are the switches of the machine that are not the lighting and
// not the fans.
type SwitchesReader interface {
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
}

// Readings are the numbers that the machine reports.
type ReadingsReader interface {
	// Instance returns the object path of the gaming functions.
	Instance() string
	// Sensors reads the live temperatures and fan speeds.
	Sensors() (Sensors, error)
	// Close releases the connection to the machine.
	Close()
}

// Gaming is the whole set of abilities of the machine.
//
// The application layer uses all of them, so it depends on this one. The
// protocol layer only needs a part of it and depends on the smaller
// interfaces instead.
type Gaming interface {
	ReadingsReader
	PowerModeReader
	GraphicsModeReader
	LightingReader
	LightingConfig
	LightingWriter
	LightingReset
	LightingBuffer
	FansReader
	FansWriter
	FanTableReader
	CoolBoostReader
	SwitchesReader
}
