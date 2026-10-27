package hardware

// Firmware is the set of abilities of the ACPI provider that the original
// service of the vendor does not cover.
//
// 原厂服务不管风扇加速、风扇表、超频能力和背光缓冲区，这几项只能在固件
// 这一侧找。这个接口就是那道缝：上层只认这个接口，不认固件的连接。
type Firmware interface {
	// CoolBoostEnabled reports whether the boost fan table is selected.
	CoolBoostEnabled() (bool, error)
	// CoolBoost switches the boost fan table.
	CoolBoost(enabled bool) error
	// FanTable reads the selected fan table.
	FanTable() (uint64, error)
	// WinKeyLocked reports whether the Windows key is locked.
	//
	// 这一项在原厂服务那边是写不动的（实测：服务收下请求，值一直不变），
	// 所以它走固件。
	WinKeyLocked() (bool, error)
	// SetWinKeyLocked locks or unlocks the Windows key.
	SetWinKeyLocked(locked bool) error
	// OverclockCapability reports which overclocking features the machine
	// supports.
	OverclockCapability() (gpu bool, cpu bool, err error)
	// BacklightBuffer reads the raw buffer of the keyboard backlight.
	BacklightBuffer() ([]byte, error)
	// ResetLighting restores the default keyboard lighting.
	ResetLighting() error
	// Close releases the connection.
	Close()
}

// OpenFirmware opens the connection to the ACPI provider.
//
// The provider needs the rights of an administrator, so the connection is
// made only when a setting that needs it is used.
func OpenFirmware() (Firmware, error) {
	backend, err := NewBackend()
	if err != nil {
		return nil, err
	}
	return backend, nil
}

// The provider of the firmware is the implementation of the interface.
var _ Firmware = (*Backend)(nil)
