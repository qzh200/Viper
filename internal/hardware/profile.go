package hardware

import (
	"fmt"

	"viper/internal/wmi"
)

// Profile bits of the Acer gaming profile word.
//
// The same method carries several unrelated switches, they are told apart
// by the bit position that is set.
const (
	// profileWinKeyLock selects the Windows key lock.
	profileWinKeyLock = 2
	// profileLCDOverdrive selects the display overdrive.
	profileLCDOverdrive = 16
	// profileOverdriveFlag marks the overdrive switch as enabled.
	profileOverdriveFlag = 1 << 48
)

// LCDOverdrive reports whether the display overdrive is active.
func (b *Backend) LCDOverdrive() (bool, error) {
	outputs, err := b.callGaming("GetGamingProfile", map[string]wmi.Value{
		"gmInput": wmi.Uint32(0),
	})
	if err != nil {
		return false, err
	}
	raw, err := decodeUint(outputs["gmOutput"])
	if err != nil {
		return false, err
	}
	return raw&0xFF == 0 && (raw>>48)&0xFF == 1, nil
}

// SetLCDOverdrive enables or disables the display overdrive.
//
// Overdrive pushes the panel beyond its rated refresh rate, so it is only
// turned on when the caller explicitly asks for it.
func (b *Backend) SetLCDOverdrive(enabled bool) error {
	input := uint64(profileLCDOverdrive)
	if enabled {
		input |= profileOverdriveFlag
	}

	if _, err := b.callGaming("SetGamingProfile", map[string]wmi.Value{
		"gmInput": wmi.Uint64(input),
	}); err != nil {
		return fmt.Errorf("hardware: set LCD overdrive: %w", err)
	}

	current, err := b.LCDOverdrive()
	if err != nil {
		return err
	}
	if current != enabled {
		return fmt.Errorf("hardware: the provider kept LCD overdrive %v instead of %v", current, enabled)
	}
	return nil
}

// WinKeyLocked reports whether the Windows key is locked.
func (b *Backend) WinKeyLocked() (bool, error) {
	outputs, err := b.callGaming("GetGamingProfile", map[string]wmi.Value{
		"gmInput": wmi.Uint32(profileWinKeyLock),
	})
	if err != nil {
		return false, err
	}
	raw, err := decodeUint(outputs["gmOutput"])
	if err != nil {
		return false, err
	}
	return (raw>>24)&0xFF == 1, nil
}

// SetWinKeyLocked locks or unlocks the Windows key.
func (b *Backend) SetWinKeyLocked(locked bool) error {
	input := uint64(profileWinKeyLock)
	if locked {
		input |= 1 << 24
	}

	if _, err := b.callGaming("SetGamingProfile", map[string]wmi.Value{
		"gmInput": wmi.Uint64(input),
	}); err != nil {
		return fmt.Errorf("hardware: set Windows key lock: %w", err)
	}

	current, err := b.WinKeyLocked()
	if err != nil {
		return err
	}
	if current != locked {
		return fmt.Errorf("hardware: the provider kept the Windows key %v instead of %v", current, locked)
	}
	return nil
}
