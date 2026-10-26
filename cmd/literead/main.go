//go:build tools

// Command literead reads the lighting state while the keyboard is lit.
//
// The earlier readings came back empty because the keyboard was dark. The
// brightness is raised first and every candidate getter is sampled.
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

	fmt.Println("=== state as it is now ===")
	lit := collect(client, path)
	print(lit)

	fmt.Println()
	fmt.Println("=== set zone 1 to the magenta of the official application ===")
	magenta := uint64(199)<<8 | uint64(0)<<16 | uint64(135)<<24 | 1
	if _, err := client.Call(hardware.ClassGaming, "SetGamingLEDColor", path, map[string]wmi.Value{
		"gmInput": wmi.Uint64(magenta),
	}); err != nil {
		fmt.Println("  failed:", err)
	}
	after := collect(client, path)
	for key, value := range after {
		if lit[key] != value {
			fmt.Printf("  %-22s %s -> %s\n", key, lit[key], value)
		}
	}

	fmt.Println()
	fmt.Println("=== set the wave effect ===")
	if _, err := client.Call(hardware.ClassGaming, "SetGamingLEDBehavior", path, map[string]wmi.Value{
		"gmInput": wmi.Uint64(3<<16 | 5<<8 | 1),
	}); err != nil {
		fmt.Println("  failed:", err)
	}
	wave := collect(client, path)
	for key, value := range wave {
		if after[key] != value {
			fmt.Printf("  %-22s %s -> %s\n", key, after[key], value)
		}
	}
}

// collect reads every candidate that could carry the lighting state.
func collect(client *wmi.Client, path string) map[string]string {
	out := make(map[string]string)
	for zone := uint32(0); zone <= 4; zone++ {
		if value, err := readRaw(client, path, "GetGamingLED", zone); err == nil {
			out[fmt.Sprintf("led(%d)", zone)] = value
		}
	}
	for query := uint32(0); query <= 8; query++ {
		if value, err := readRaw(client, path, "GetGamingLEDColor", query); err == nil {
			out[fmt.Sprintf("colour(%d)", query)] = value
		}
	}
	for query := uint32(0); query <= 8; query++ {
		if value, err := readRaw(client, path, "GetGamingLEDBehavior", query); err == nil {
			out[fmt.Sprintf("behaviour(%d)", query)] = value
		}
	}
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

// print shows the collected readings.
func print(values map[string]string) {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		fmt.Printf("  %-16s %s\n", key, values[key])
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
		if value.Kind == wmi.KindBytes {
			parts = append(parts, name+"=["+hexBytes(value.Bytes)+"]")
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
