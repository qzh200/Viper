//go:build tools

// Command rgbprobe looks for a way to read the keyboard lighting back.
//
// The colour and behaviour getters do not report anything useful, so the
// remaining candidates are compared before and after the lighting is
// changed.
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

	before := collect(client, path)
	fmt.Println("=== before ===")
	print(before)

	fmt.Println()
	fmt.Println("=== set every zone to red, static ===")
	for zone := uint64(1); zone <= 4; zone++ {
		colour := uint64(0xFF)<<8 | zone
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
	if _, err := client.Call(hardware.ClassGaming, "SetGamingRgbKb", path, map[string]wmi.Value{
		"gmInput": wmi.Uint64(1 | 100<<8),
	}); err != nil {
		fmt.Println("  rgb kb ->", err)
	}

	after := collect(client, path)
	fmt.Println()
	fmt.Println("=== differences ===")
	for key, value := range after {
		if before[key] != value {
			fmt.Printf("  %s: %s -> %s\n", key, before[key], value)
		}
	}

	fmt.Println()
	fmt.Println("=== restore ===")
	for zone := uint64(1); zone <= 4; zone++ {
		colour := uint64(0x00)<<8 | zone
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
}

// collect reads every candidate that could carry the lighting state.
func collect(client *wmi.Client, path string) map[string]string {
	out := make(map[string]string)

	for query := uint32(0); query <= 8; query++ {
		if value, err := readRaw(client, path, "GetGamingRgbKb", query); err == nil {
			out[fmt.Sprintf("rgbKb(%d)", query)] = value
		}
	}
	for query := uint32(0); query <= 16; query++ {
		if value, err := readRaw(client, path, "GetGamingMiscSetting", query); err == nil {
			out[fmt.Sprintf("misc(%d)", query)] = value
		}
	}
	for query := uint32(0); query <= 8; query++ {
		if value, err := readRaw(client, path, "GetGamingProfileSetting", query); err == nil {
			out[fmt.Sprintf("profileSetting(%d)", query)] = value
		}
	}
	for query := uint32(0); query <= 8; query++ {
		if value, err := readRaw(client, path, "GetGamingProfile", query); err == nil {
			out[fmt.Sprintf("profile(%d)", query)] = value
		}
	}
	return out
}

// print shows the collected readings.
func print(values map[string]string) {
	for key, value := range values {
		fmt.Printf("  %-22s %s\n", key, value)
	}
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
		parts = append(parts, name+"="+value.String())
	}
	if len(parts) == 0 {
		return "", fmt.Errorf("no output")
	}
	return strings.Join(parts, " "), nil
}
