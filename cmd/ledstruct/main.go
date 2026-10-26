//go:build tools

// Command ledstruct reveals the layout of the lighting configuration.
//
// The keyboard is switched on with the interfaces that are known to work,
// then GetGamingLED is read back so that the meaning of every byte of the
// structure becomes visible.
package main

import (
	"fmt"
	"os"
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

	fmt.Println("=== before ===")
	dumpZones(client, path)

	fmt.Println()
	fmt.Println("=== set a static red colour in every zone ===")
	for zone := uint64(1); zone <= 4; zone++ {
		// The colour word is blue, green, red, zone.
		colour := uint64(0x00)<<24 | uint64(0x00)<<16 | uint64(0xFF)<<8 | zone
		if _, err := client.Call(hardware.ClassGaming, "SetGamingLEDColor", path, map[string]wmi.Value{
			"gmInput": wmi.Uint64(colour),
		}); err != nil {
			fmt.Printf("  zone %d -> %v\n", zone, err)
		}
	}
	// Effect, speed and direction.
	if _, err := client.Call(hardware.ClassGaming, "SetGamingLEDBehavior", path, map[string]wmi.Value{
		"gmInput": wmi.Uint64(0<<16 | 3<<8 | 0),
	}); err != nil {
		fmt.Println("  behaviour ->", err)
	}
	fmt.Println()
	dumpZones(client, path)

	fmt.Println()
	fmt.Println("=== set a wave effect ===")
	if _, err := client.Call(hardware.ClassGaming, "SetGamingLEDBehavior", path, map[string]wmi.Value{
		"gmInput": wmi.Uint64(2<<16 | 5<<8 | 1),
	}); err != nil {
		fmt.Println("  behaviour ->", err)
	}
	fmt.Println()
	dumpZones(client, path)

	fmt.Println()
	fmt.Println("=== restore: colours off, static effect ===")
	for zone := uint64(1); zone <= 4; zone++ {
		colour := uint64(0x00)<<24 | uint64(0x00)<<16 | uint64(0x00)<<8 | zone
		if _, err := client.Call(hardware.ClassGaming, "SetGamingLEDColor", path, map[string]wmi.Value{
			"gmInput": wmi.Uint64(colour),
		}); err != nil {
			fmt.Printf("  zone %d -> %v\n", zone, err)
		}
	}
	if _, err := client.Call(hardware.ClassGaming, "SetGamingLEDBehavior", path, map[string]wmi.Value{
		"gmInput": wmi.Uint64(0<<16 | 3<<8 | 0),
	}); err != nil {
		fmt.Println("  behaviour ->", err)
	}
	fmt.Println()
	dumpZones(client, path)
}

// dumpZones prints the lighting structure of every zone.
func dumpZones(client *wmi.Client, path string) {
	for zone := uint32(0); zone <= 4; zone++ {
		outputs, err := client.Call(hardware.ClassGaming, "GetGamingLED", path, map[string]wmi.Value{
			"gmInput": wmi.Uint32(zone),
		})
		if err != nil {
			fmt.Printf("  zone %d -> %v\n", zone, err)
			continue
		}
		fmt.Printf("  zone %d:", zone)
		for _, name := range []string{"gmOutput", "gmReturn"} {
			value, ok := outputs[name]
			if !ok {
				continue
			}
			if value.Kind == wmi.KindBytes {
				fmt.Printf("  %s=[%s]", name, hexBytes(value.Bytes))
				continue
			}
			fmt.Printf("  %s=%s", name, value.String())
		}
		fmt.Println()
	}
}

// hexBytes renders a byte slice as hex.
func hexBytes(data []byte) string {
	parts := make([]string, 0, len(data))
	for _, b := range data {
		parts = append(parts, fmt.Sprintf("%02X", b))
	}
	return strings.Join(parts, " ")
}
