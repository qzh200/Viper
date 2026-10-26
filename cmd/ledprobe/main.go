//go:build tools

// Command ledprobe dumps the raw structures that describe the keyboard
// lighting and the fans.
//
// GetGamingLED returns the full lighting configuration of a zone as a byte
// buffer, which is what the Pulsar keyboards are configured with.
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

	fmt.Println("=== GetGamingLED per zone ===")
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

	fmt.Println()
	fmt.Println("=== lighting behaviour and colour ===")
	for _, method := range []string{"GetGamingLEDBehavior", "GetGamingLEDColor"} {
		for query := uint32(0); query <= 4; query++ {
			value, err := readRaw(client, path, method, query)
			if err != nil {
				continue
			}
			fmt.Printf("  %s(%d) = %s\n", method, query, value)
		}
	}

	fmt.Println()
	fmt.Println("=== fan state ===")
	for _, probe := range []struct {
		method string
		query  uint32
	}{
		{"GetGamingFanTable", 0},
		{"GetGamingFanBehavior", 0},
		{"GetGamingFanBehavior", 1},
		{"GetGamingFanBehavior", 2},
		{"GetGamingFanBehavior", 3},
		{"GetGamingFanBehavior", 4},
		{"GetGamingFanSpeed", 1},
		{"GetGamingFanSpeed", 2},
		{"GetGamingFanSpeed", 3},
		{"GetGamingFanSpeed", 4},
		{"GetGamingFanSpeed", 5},
	} {
		value, err := readRaw(client, path, probe.method, probe.query)
		if err != nil {
			fmt.Printf("  %s(%d) -> %v\n", probe.method, probe.query, err)
			continue
		}
		fmt.Printf("  %s(%d) = %s\n", probe.method, probe.query, value)
	}
}

// readRaw invokes a method and renders the raw output value.
func readRaw(client *wmi.Client, path, method string, query uint32) (string, error) {
	outputs, err := client.Call(hardware.ClassGaming, method, path, map[string]wmi.Value{
		"gmInput": wmi.Uint32(query),
	})
	if err != nil {
		return "", err
	}
	parts := make([]string, 0, len(outputs))
	for _, name := range []string{"gmOutput", "gmReturn"} {
		value, ok := outputs[name]
		if !ok {
			continue
		}
		if value.Kind == wmi.KindBytes {
			parts = append(parts, fmt.Sprintf("%s=[%s]", name, hexBytes(value.Bytes)))
			continue
		}
		parts = append(parts, name+"="+value.String())
	}
	if len(parts) == 0 {
		return "", fmt.Errorf("no output")
	}
	return strings.Join(parts, " "), nil
}

// hexBytes renders a byte slice as hex.
func hexBytes(data []byte) string {
	parts := make([]string, 0, len(data))
	for _, b := range data {
		parts = append(parts, fmt.Sprintf("%02X", b))
	}
	return strings.Join(parts, " ")
}
