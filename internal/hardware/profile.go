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
)

// WinKeyLocked reports whether the Windows key is locked.
func (b *Backend) WinKeyLocked() (bool, error) {
	outputs, err := b.callGaming("GetGamingProfile", map[string]wmi.Value{
		"gmInput": wmi.Uint32(profileWinKeyLock),
	})
	if err != nil {
		return false, err
	}
	// 固件没有给出 gmOutput 时不能当成读到了零，那会让上层把"读不到"当成
	// "Windows 键没有锁"。
	value, ok := outputs["gmOutput"]
	if !ok {
		return false, fmt.Errorf("hardware: 读取档位时固件没有给出 gmOutput")
	}
	raw, err := decodeUint(value)
	if err != nil {
		return false, err
	}
	return (raw>>24)&0xFF == 1, nil
}

// SetWinKeyLocked locks or unlocks the Windows key.
func (b *Backend) SetWinKeyLocked(locked bool) error {
	_, err := operate(b, func() (struct{}, error) {
		input := uint64(profileWinKeyLock)
		if locked {
			input |= 1 << 24
		}

		if _, err := b.callGaming("SetGamingProfile", map[string]wmi.Value{
			"gmInput": wmi.Uint64(input),
		}); err != nil {
			return struct{}{}, fmt.Errorf("hardware: set Windows key lock: %w", err)
		}

		current, err := b.WinKeyLocked()
		if err != nil {
			return struct{}{}, err
		}
		if current != locked {
			return struct{}{}, fmt.Errorf("hardware: the provider kept the Windows key %v instead of %v", current, locked)
		}
		return struct{}{}, nil
	})
	return err
}
