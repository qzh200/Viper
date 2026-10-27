//go:build tools

// Command coverage looks for the parts of the provider that the hardware
// layer does not use yet.
//
// Every call of the tool reads, nothing is written, so the machine keeps
// the state it has.
package main

import (
	"fmt"
	"os"

	"viper/internal/hardware"
	"viper/internal/wmi"
)

func main() {
	client, gaming, action := connect()

	fmt.Println("=== GetGamingSysInfo, every category with index one ===")
	for category := uint32(0); category <= 0x10; category++ {
		query := category<<8 | 1
		outputs, err := client.Call(hardware.ClassGaming, "GetGamingSysInfo", gaming, map[string]wmi.Value{
			"gmInput": wmi.Uint32(query),
		})
		if err != nil {
			fmt.Printf("  category 0x%02X -> %v\n", category, err)
			continue
		}
		value, ok := outputs["gmOutput"]
		if !ok {
			continue
		}
		fmt.Printf("  category 0x%02X -> %s\n", category, value.String())
	}

	fmt.Println("\n=== GetGamingProfileSetting, every query ===")
	for query := uint32(0); query <= 8; query++ {
		outputs, err := client.Call(hardware.ClassGaming, "GetGamingProfileSetting", gaming, map[string]wmi.Value{
			"gmInput": wmi.Uint32(query),
		})
		if err != nil {
			fmt.Printf("  query %d -> %v\n", query, err)
			continue
		}
		value, ok := outputs["gmOutput"]
		if !ok {
			continue
		}
		fmt.Printf("  query %d -> %s\n", query, value.String())
	}

	fmt.Println("\n=== GetGamingProfile, every query ===")
	for query := uint32(0); query <= 8; query++ {
		outputs, err := client.Call(hardware.ClassGaming, "GetGamingProfile", gaming, map[string]wmi.Value{
			"gmInput": wmi.Uint32(query),
		})
		if err != nil {
			fmt.Printf("  query %d -> %v\n", query, err)
			continue
		}
		value, ok := outputs["gmOutput"]
		if !ok {
			continue
		}
		fmt.Printf("  query %d -> %s\n", query, value.String())
	}

	fmt.Println("\n=== APGeAction GetFunction, the hotkey numbers ===")
	// The backlight uses the hotkey 132 and the boost the number 7. The
	// other hotkeys of the keyboard controller are looked for here. The
	// two services use a different shape of the query, so both are tried.
	for _, shape := range []struct {
		name string
		make func(hotkey uint32) uint64
	}{
		{"backlight shape", func(hotkey uint32) uint64 { return uint64(hotkey)<<8 | 0x80001 }},
		{"boost shape", func(hotkey uint32) uint64 { return uint64(hotkey)<<8 | 0x0200 }},
	} {
		fmt.Printf("--- %s ---\n", shape.name)
		for hotkey := uint32(0); hotkey <= 200; hotkey++ {
			query := shape.make(hotkey)
			outputs, err := client.Call(hardware.ClassAction, "GetFunction", action, map[string]wmi.Value{
				"uiInput": wmi.Uint32(uint32(query)),
			})
			if err != nil {
				continue
			}
			value, ok := outputs["uiOutput"]
			if !ok {
				continue
			}
			raw := value.UInt64()
			// The answer of an unknown function is a fixed error code.
			if raw == 0xE2 {
				continue
			}
			fmt.Printf("  hotkey %3d -> 0x%08X  byte1=%d byte2=%d byte4=%d byte5=%d\n",
				hotkey, raw, (raw>>8)&0xFF, (raw>>16)&0xFF, (raw>>32)&0xFF, (raw>>40)&0xFF)
		}
	}
}

// connect opens the ACPI namespace and returns the instance paths.
func connect() (*wmi.Client, string, string) {
	client, err := wmi.Connect(wmi.NamespaceACPI)
	if err != nil {
		fmt.Println("wmi:", err)
		os.Exit(1)
	}
	gamingPaths, err := client.Instances(hardware.ClassGaming)
	if err != nil || len(gamingPaths) == 0 {
		fmt.Println("no gaming instance:", err)
		os.Exit(1)
	}
	actionPaths, err := client.Instances(hardware.ClassAction)
	if err != nil || len(actionPaths) == 0 {
		return client, gamingPaths[0], gamingPaths[0]
	}
	return client, gamingPaths[0], actionPaths[0]
}
