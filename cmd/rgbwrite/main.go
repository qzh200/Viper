//go:build tools

// Command rgbwrite finds out which call really changes the keyboard
// brightness.
//
// The lighting is owned by the service of the original software, so a
// value that is written may be taken over again a moment later. The
// probe writes a level with every known path and reads it back right
// away and after a while.
package main

import (
	"fmt"
	"os"
	"time"

	"viper/internal/hardware"
	"viper/internal/wmi"
)

func main() {
	client, path := connect()

	fmt.Println("=== baseline ===")
	fmt.Printf("  brightness = %v\n", brightness(client, path))

	// Every candidate for writing the level.
	candidates := []struct {
		name string
		run  func() error
	}{
		{"SetGamingRgbKb device/level", func() error {
			_, err := client.Call(hardware.ClassGaming, "SetGamingRgbKb", path, map[string]wmi.Value{
				"gmInput": wmi.Uint64(1 | 42<<8),
			})
			return err
		}},
		{"SetGamingRgbKb level/device", func() error {
			_, err := client.Call(hardware.ClassGaming, "SetGamingRgbKb", path, map[string]wmi.Value{
				"gmInput": wmi.Uint64(42 | 1<<8),
			})
			return err
		}},
		{"SetGamingRgbKb plain 42", func() error {
			_, err := client.Call(hardware.ClassGaming, "SetGamingRgbKb", path, map[string]wmi.Value{
				"gmInput": wmi.Uint64(42),
			})
			return err
		}},
		{"SetGamingKBBacklight buffer", func() error {
			buffer := make([]byte, 16)
			for i := 0; i < 8; i++ {
				buffer[i] = byte(42 << (8 * i))
			}
			buffer[9] = 1
			_, err := client.Call(hardware.ClassGaming, "SetGamingKBBacklight", path, map[string]wmi.Value{
				"gmInput": wmi.Bytes(buffer),
			})
			return err
		}},
		{"APGeAction SetFunction", func() error {
			input := uint64(2) | 132<<8 | 0x80000 | 42<<32
			_, err := client.Call(hardware.ClassAction, "SetFunction", path, map[string]wmi.Value{
				"uiInput": wmi.Uint64(input),
			})
			return err
		}},
	}

	for _, candidate := range candidates {
		fmt.Printf("\n=== %s ===\n", candidate.name)
		if err := candidate.run(); err != nil {
			fmt.Printf("  write failed: %v\n", err)
			continue
		}
		fmt.Printf("  immediately   : %v\n", brightness(client, path))
		for _, wait := range []time.Duration{500 * time.Millisecond, 2 * time.Second, 5 * time.Second} {
			time.Sleep(wait)
			fmt.Printf("  after %-6v : %v\n", wait, brightness(client, path))
		}
	}

	fmt.Println("\n=== restore brightness 100 ===")
	_, _ = client.Call(hardware.ClassGaming, "SetGamingRgbKb", path, map[string]wmi.Value{
		"gmInput": wmi.Uint64(1 | 100<<8),
	})
	time.Sleep(time.Second)
	fmt.Printf("  brightness = %v\n", brightness(client, path))
}

// brightness reads the level of the keyboard.
func brightness(client *wmi.Client, path string) string {
	outputs, err := client.Call(hardware.ClassGaming, "GetGamingRgbKb", path, map[string]wmi.Value{
		"gmInput": wmi.Uint32(1),
	})
	if err != nil {
		return "error: " + err.Error()
	}
	value, ok := outputs["gmOutput"]
	if !ok {
		return "no output"
	}
	return value.String()
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
