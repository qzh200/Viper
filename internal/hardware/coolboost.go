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
	input := uint64(7)
	if enabled {
		input |= 1 << 16
	}
	if _, err := b.callAction("SetFunction", map[string]wmi.Value{
		"uiInput": wmi.Uint64(input),
	}); err != nil {
		return fmt.Errorf("hardware: set CoolBoost: %w", err)
	}
	return nil
}

// CoolBoostEnabled reports whether CoolBoost is active.
func (b *Backend) CoolBoostEnabled() (bool, error) {
	outputs, err := b.callAction("GetFunction", map[string]wmi.Value{
		"uiInput": wmi.Uint32(0x0207),
	})
	if err != nil {
		return false, err
	}
	raw, err := decodeUint(outputs["uiOutput"])
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
