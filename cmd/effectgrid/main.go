//go:build tools

// Command effectgrid maps the accepted combinations of the lighting
// parameters.
//
// A call only fails when the provider rejects the whole word, so the
// effect, speed and direction are swept together.
package main

import (
	"fmt"
	"os"
	"strings"

	"viper/internal/hardware"
	"viper/internal/wmi"
)

func main() {
	client, path := connect()

	fmt.Println("=== effect x speed at direction 0 ===")
	for effect := uint64(0); effect <= 7; effect++ {
		var accepted []uint64
		for speed := uint64(0); speed <= 8; speed++ {
			if err := set(client, path, effect, speed, 0); err == nil {
				accepted = append(accepted, speed)
			}
		}
		fmt.Printf("  effect %d: speeds %v\n", effect, accepted)
	}

	fmt.Println()
	fmt.Println("=== effect x direction at speed 3 ===")
	for effect := uint64(0); effect <= 7; effect++ {
		var accepted []uint64
		for direction := uint64(0); direction <= 3; direction++ {
			if err := set(client, path, effect, 3, direction); err == nil {
				accepted = append(accepted, direction)
			}
		}
		fmt.Printf("  effect %d: directions %v\n", effect, accepted)
	}

	fmt.Println()
	fmt.Println("=== the wave combination from the screenshot ===")
	for _, combo := range [][3]uint64{
		{3, 5, 1}, {3, 5, 0}, {3, 3, 1}, {3, 1, 1}, {2, 5, 1},
	} {
		err := set(client, path, combo[0], combo[1], combo[2])
		fmt.Printf("  effect %d speed %d direction %d -> %s\n",
			combo[0], combo[1], combo[2], verdict(err))
	}

	fmt.Println()
	fmt.Println("=== restore ===")
	set(client, path, 0, 3, 0)
	for zone := uint64(1); zone <= 4; zone++ {
		client.Call(hardware.ClassGaming, "SetGamingLEDColor", path, map[string]wmi.Value{
			"gmInput": wmi.Uint64(zone),
		})
	}
	client.Call(hardware.ClassGaming, "SetGamingRgbKb", path, map[string]wmi.Value{
		"gmInput": wmi.Uint64(1),
	})
}

// set writes the lighting behaviour.
func set(client *wmi.Client, path string, effect, speed, direction uint64) error {
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

// keep strings referenced.
var _ = strings.TrimSpace
