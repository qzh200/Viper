//go:build tools

// Command effectscan finds the accepted lighting parameters.
//
// The provider rejects unknown values with WBEM_E_INVALID_PARAMETER, so
// every candidate effect, speed and direction can be told apart by
// whether the call is accepted.
package main

import (
	"fmt"
	"os"

	"viper/internal/hardware"
	"viper/internal/wmi"
)

func main() {
	client, path := connect()

	fmt.Println("=== effect values ===")
	for effect := uint64(0); effect <= 15; effect++ {
		err := setBehaviour(client, path, effect, 3, 0)
		fmt.Printf("  effect %2d -> %s\n", effect, verdict(err))
	}

	fmt.Println()
	fmt.Println("=== speed values ===")
	for speed := uint64(0); speed <= 8; speed++ {
		err := setBehaviour(client, path, 0, speed, 0)
		fmt.Printf("  speed %2d -> %s\n", speed, verdict(err))
	}

	fmt.Println()
	fmt.Println("=== direction values ===")
	for direction := uint64(0); direction <= 3; direction++ {
		err := setBehaviour(client, path, 0, 3, direction)
		fmt.Printf("  direction %2d -> %s\n", direction, verdict(err))
	}

	fmt.Println()
	fmt.Println("=== colour zones ===")
	for zone := uint64(1); zone <= 5; zone++ {
		colour := uint64(0xFF)<<8 | zone
		_, err := client.Call(hardware.ClassGaming, "SetGamingLEDColor", path, map[string]wmi.Value{
			"gmInput": wmi.Uint64(colour),
		})
		fmt.Printf("  zone %d -> %s\n", zone, verdict(err))
	}

	fmt.Println()
	fmt.Println("=== brightness values ===")
	for level := uint64(0); level <= 100; level += 25 {
		_, err := client.Call(hardware.ClassGaming, "SetGamingRgbKb", path, map[string]wmi.Value{
			"gmInput": wmi.Uint64(1 | level<<8),
		})
		fmt.Printf("  level %3d -> %s\n", level, verdict(err))
	}

	fmt.Println()
	fmt.Println("=== restore ===")
	setBehaviour(client, path, 0, 3, 0)
	for zone := uint64(1); zone <= 4; zone++ {
		client.Call(hardware.ClassGaming, "SetGamingLEDColor", path, map[string]wmi.Value{
			"gmInput": wmi.Uint64(zone),
		})
	}
	client.Call(hardware.ClassGaming, "SetGamingRgbKb", path, map[string]wmi.Value{
		"gmInput": wmi.Uint64(1),
	})
}

// setBehaviour writes the lighting behaviour.
func setBehaviour(client *wmi.Client, path string, effect, speed, direction uint64) error {
	_, err := client.Call(hardware.ClassGaming, "SetGamingLEDBehavior", path, map[string]wmi.Value{
		"gmInput": wmi.Uint64(effect<<16 | speed<<8 | direction),
	})
	return err
}

// verdict renders the outcome of a call.
func verdict(err error) string {
	if err == nil {
		return "accepted"
	}
	return "rejected: " + err.Error()
}

// connect opens the ACPI namespace and returns the gaming instance path.
func connect() (*wmi.Client, string) {
	client, err := wmi.Connect(wmi.NamespaceACPI)
	if err != nil {
		fmt.Println("wmi:", err)
		os.Exit(1)
	}
	paths, err := client.Instances(hardware.ClassGaming)
	if err != nil || len(paths) == 0 {
		fmt.Println("no instance:", err)
		os.Exit(1)
	}
	return client, paths[0]
}
