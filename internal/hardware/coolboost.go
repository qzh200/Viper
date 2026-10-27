package hardware

import (
	"fmt"

	"viper/internal/wmi"
)

// CoolBoost enables or disables the boost fan table.
//
// CoolBoost lives on the generic ACPI action class rather than the Acer
// gaming class, so it is driven through APGeAction.
func (b *Backend) CoolBoost(enabled bool) error {
	_, err := operate(b, func() (struct{}, error) {
		input := uint64(7)
		if enabled {
			input |= 1 << 16
		}
		if _, err := b.callAction("SetFunction", map[string]wmi.Value{
			"uiInput": wmi.Uint64(input),
		}); err != nil {
			return struct{}{}, fmt.Errorf("hardware: set CoolBoost: %w", err)
		}
		return struct{}{}, nil
	})
	return err
}

// CoolBoostEnabled reports whether CoolBoost is active.
func (b *Backend) CoolBoostEnabled() (bool, error) {
	outputs, err := b.callAction("GetFunction", map[string]wmi.Value{
		"uiInput": wmi.Uint32(0x0207),
	})
	if err != nil {
		return false, err
	}
	// 固件没有给出 uiOutput 时不能当成读到了零，那会让上层把"读不到"当成
	// "加速是关着的"。
	value, ok := outputs["uiOutput"]
	if !ok {
		return false, fmt.Errorf("hardware: 读取加速状态时固件没有给出 uiOutput")
	}
	raw, err := decodeUint(value)
	if err != nil {
		return false, err
	}
	if raw&0xFF == 0 && raw > 0 {
		return (raw>>8)&0xFF == 1, nil
	}

	// The action class did not answer, fall back to the fan table.
	outputs, err = b.callGaming("GetGamingFanTable", nil)
	if err != nil {
		return false, err
	}
	table, err := decodeUint(outputs["gmOutput"])
	if err != nil {
		return false, err
	}
	return table&0xFF == 0 && (table>>8)&0xFF == FanTableBoost, nil
}
