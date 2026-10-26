//go:build tools

// Command ledformat searches for the layout of the SetGamingLED
// structure.
//
// SetGamingLED takes twelve bytes, so the first byte of every candidate
// layout is swept and the answer is looked for in the structure that
// GetGamingLED reports back.
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

	fmt.Println("=== baseline ===")
	dump(client, path)

	// The layout of the keyboard backlight is known: the value fills the
	// first eight bytes, byte eight is zero, byte nine is one and the
	// last two bytes are zero. The lighting structure is very likely
	// built the same way, so the candidates below follow that pattern.
	type candidate struct {
		name string
		data []byte
	}
	candidates := []candidate{
		{"zone1 red static", []byte{0x01, 0x00, 0x00, 0x00, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00}},
		{"zone1 red static v2", []byte{0x01, 0x00, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00}},
		{"all red static", []byte{0x00, 0x00, 0x00, 0x00, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00}},
		{"zone1 effect speed", []byte{0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00}},
		{"zone1 wave speed 3", []byte{0x01, 0x02, 0x03, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00}},
		{"zone1 wave speed 3 v2", []byte{0x01, 0x02, 0x03, 0x00, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00}},
		{"zone1 white brightness", []byte{0x01, 0x00, 0x00, 0x64, 0xFF, 0xFF, 0xFF, 0x00, 0x00, 0x01, 0x00, 0x00}},
	}

	for _, c := range candidates {
		fmt.Printf("=== %s ===\n", c.name)
		if _, err := client.Call(hardware.ClassGaming, "SetGamingLED", path, map[string]wmi.Value{
			"gmInput": wmi.Bytes(c.data),
		}); err != nil {
			fmt.Println("  failed:", err)
		}
		dump(client, path)
		fmt.Println()
	}

	fmt.Println("=== restore ===")
	if _, err := client.Call(hardware.ClassGaming, "SetGamingLED", path, map[string]wmi.Value{
		"gmInput": wmi.Bytes(make([]byte, 12)),
	}); err != nil {
		fmt.Println("  failed:", err)
	}
	dump(client, path)
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

// dump prints the lighting structure of every zone.
func dump(client *wmi.Client, path string) {
	for zone := uint32(0); zone <= 2; zone++ {
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
