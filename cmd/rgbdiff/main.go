//go:build tools

// Command rgbdiff separates the three lighting properties.
//
// The colour, the effect and the brightness are changed one at a time and
// every candidate reading is compared, so that each property is matched
// with the getter that reports it.
package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"viper/internal/hardware"
	"viper/internal/wmi"
)

func main() {
	client, path := connect()

	base := collect(client, path)

	fmt.Println("=== change only the colour of zone 1 to green ===")
	setColour(client, path, 1, 0x00, 0xFF, 0x00)
	report(base, collect(client, path))

	fmt.Println()
	fmt.Println("=== change only the effect to wave ===")
	setEffect(client, path, 2, 5, 1)
	report(base, collect(client, path))

	fmt.Println()
	fmt.Println("=== change only the brightness to 60 ===")
	setBrightness(client, path, 60)
	report(base, collect(client, path))

	fmt.Println()
	fmt.Println("=== restore ===")
	setColour(client, path, 1, 0, 0, 0)
	setEffect(client, path, 0, 3, 0)
	setBrightness(client, path, 0)
}

// setColour writes one zone colour.
func setColour(client *wmi.Client, path string, zone uint64, r, g, b uint64) {
	colour := (b << 24) | (g << 16) | (r << 8) | zone
	if _, err := client.Call(hardware.ClassGaming, "SetGamingLEDColor", path, map[string]wmi.Value{
		"gmInput": wmi.Uint64(colour),
	}); err != nil {
		fmt.Println("  colour ->", err)
	}
}

// setEffect writes the lighting behaviour.
func setEffect(client *wmi.Client, path string, effect, speed, direction uint64) {
	if _, err := client.Call(hardware.ClassGaming, "SetGamingLEDBehavior", path, map[string]wmi.Value{
		"gmInput": wmi.Uint64(effect<<16 | speed<<8 | direction),
	}); err != nil {
		fmt.Println("  effect ->", err)
	}
}

// setBrightness writes the keyboard brightness.
func setBrightness(client *wmi.Client, path string, level uint64) {
	if _, err := client.Call(hardware.ClassGaming, "SetGamingRgbKb", path, map[string]wmi.Value{
		"gmInput": wmi.Uint64(1 | level<<8),
	}); err != nil {
		fmt.Println("  brightness ->", err)
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
	for query := uint32(0); query <= 20; query++ {
		if value, err := readRaw(client, path, "GetGamingMiscSetting", query); err == nil {
			out[fmt.Sprintf("misc(%d)", query)] = value
		}
	}
	return out
}

// report prints the readings that changed.
func report(before, after map[string]string) {
	keys := make([]string, 0, len(after))
	for key := range after {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if before[key] != after[key] {
			fmt.Printf("  %-12s %s -> %s\n", key, before[key], after[key])
		}
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
