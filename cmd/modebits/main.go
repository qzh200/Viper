//go:build windows

// Command modebits prints the full raw value of the operation mode for a
// range of encodings, and reports the machine model taken from SMBIOS.
//
// The raw value matters because the provider packs more than one field into
// it: a narrow mask hides the turbo bit.
package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"viper/internal/hardware"
	"viper/internal/wmi"
)

func main() {
	backend, err := hardware.NewBackend()
	if err != nil {
		fmt.Println("hardware:", err)
		os.Exit(1)
	}
	defer backend.Close()

	client, err := wmi.Connect(wmi.NamespaceACPI)
	if err != nil {
		fmt.Println("wmi:", err)
		os.Exit(1)
	}
	defer client.Close()

	paths, err := client.Instances(hardware.ClassGaming)
	if err != nil || len(paths) == 0 {
		fmt.Println("no instance:", err)
		os.Exit(1)
	}
	path := paths[0]

	fmt.Println("=== raw operation mode per encoding ===")
	for value := 0; value <= 20; value++ {
		if _, err := client.Call(hardware.ClassGaming, "SetGamingMiscSetting", path, map[string]wmi.Value{
			"gmInput": wmi.Uint64(uint64(value)<<8 | 0x0B),
		}); err != nil {
			fmt.Printf("  value %2d -> call failed: %v\n", value, err)
			continue
		}
		raw, err := raw(client, path)
		if err != nil {
			fmt.Printf("  value %2d -> read failed: %v\n", value, err)
			continue
		}
		fmt.Printf("  value %2d -> raw 0x%016X  byte1=0x%02X\n", value, raw, (raw>>8)&0xFF)
	}

	fmt.Println()
	fmt.Println("=== machine identity from SMBIOS ===")
	model, family, product := hardware.SystemIdentity()
	fmt.Printf("  product : %s\n", product)
	fmt.Printf("  family  : %s\n", family)
	fmt.Printf("  model   : %s\n", model)
	fmt.Printf("  isPredator: %v\n", hardware.IsPredatorModel(model, family))

	fmt.Println()
	fmt.Println("=== restore balanced ===")
	if err := backend.SetPowerMode(hardware.PowerBalanced); err != nil {
		fmt.Println("  failed:", err)
	} else {
		mode, _ := backend.PowerMode()
		fmt.Println("  mode is now", mode)
	}
}

// raw reads the full operation mode value.
func raw(client *wmi.Client, path string) (uint64, error) {
	outputs, err := client.Call(hardware.ClassGaming, "GetGamingMiscSetting", path, map[string]wmi.Value{
		"gmInput": wmi.Uint32(0x0B),
	})
	if err != nil {
		return 0, err
	}
	value, ok := outputs["gmOutput"]
	if !ok {
		return 0, fmt.Errorf("no gmOutput")
	}
	text := strings.TrimSpace(value.String())
	base := 10
	if strings.HasPrefix(text, "0x") || strings.HasPrefix(text, "0X") {
		text = text[2:]
		base = 16
	}
	n, err := strconv.ParseUint(text, base, 64)
	if err != nil {
		return 0, err
	}
	return n, nil
}
