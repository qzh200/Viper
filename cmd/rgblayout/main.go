//go:build tools

// Command rgblayout finds out the layout of the keyboard backlight buffer
// and looks for a way to read the lighting back.
//
// The provider offers a getter for the backlight buffer and a getter for
// the lighting, both are compared before and after a change.
package main

import (
	"encoding/hex"
	"fmt"
	"os"
	"strings"

	"viper/internal/hardware"
	"viper/internal/wmi"
)

func main() {
	client, path := connect()

	fmt.Println("=== baseline ===")
	fmt.Printf("  rgbKb        : %s\n", raw(client, path, "GetGamingRgbKb", 1))
	fmt.Printf("  kbBacklight  : %s\n", buffer(client, path, "GetGamingKBBacklight", 0))
	fmt.Printf("  kbBacklight1 : %s\n", buffer(client, path, "GetGamingKBBacklight", 1))
	fmt.Printf("  led(0)       : %s\n", buffer(client, path, "GetGamingLED", 0))
	fmt.Printf("  led(1)       : %s\n", buffer(client, path, "GetGamingLED", 1))
	fmt.Printf("  ledBehavior  : %s\n", raw(client, path, "GetGamingLEDBehavior", 0))
	fmt.Printf("  ledColor     : %s\n", raw(client, path, "GetGamingLEDColor", 0))

	fmt.Println("\n=== backlight buffer candidates ===")
	candidates := []struct {
		name string
		data []byte
	}{
		{"level in byte 0", bufferFor(42, 0)},
		{"level in byte 1", bufferFor(42, 1)},
		{"level in byte 2", bufferFor(42, 2)},
		{"level in byte 0, byte9=1", bufferFor(42, 0)},
		{"marker bytes", []byte{0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x00, 0x01, 0x00, 0x00}},
	}
	for _, candidate := range candidates {
		fmt.Printf("--- %s ---\n", candidate.name)
		writeBuffer(client, path, candidate.data)
		fmt.Printf("  rgbKb        : %s\n", raw(client, path, "GetGamingRgbKb", 1))
		fmt.Printf("  kbBacklight  : %s\n", buffer(client, path, "GetGamingKBBacklight", 0))
	}

	fmt.Println("\n=== zone colours ===")
	// Magenta in zone one, green in zone two.
	writeColour(client, path, 0xC7, 0x00, 0x87, 1)
	writeColour(client, path, 0x00, 0xFF, 0x00, 2)
	for zone := uint32(0); zone <= 4; zone++ {
		fmt.Printf("  led(%d)       : %s\n", zone, buffer(client, path, "GetGamingLED", zone))
	}
	fmt.Printf("  ledColor     : %s\n", raw(client, path, "GetGamingLEDColor", 0))
	fmt.Printf("  ledBehavior  : %s\n", raw(client, path, "GetGamingLEDBehavior", 0))

	fmt.Println("\n=== effect candidates ===")
	for effect := uint64(0); effect <= 7; effect++ {
		writeBehaviour(client, path, effect, 5, 1)
		fmt.Printf("  effect %d -> behavior %s | led %s\n", effect,
			raw(client, path, "GetGamingLEDBehavior", 0),
			buffer(client, path, "GetGamingLED", 0))
	}

	fmt.Println("\n=== restore ===")
	writeColour(client, path, 0x00, 0xFF, 0xC7, 1)
	writeColour(client, path, 0x00, 0xFF, 0xC7, 2)
	writeColour(client, path, 0x00, 0xFF, 0xC7, 3)
	writeColour(client, path, 0x00, 0xFF, 0xC7, 4)
	writeBehaviour(client, path, 0, 5, 0)
	_, _ = client.Call(hardware.ClassGaming, "SetGamingRgbKb", path, map[string]wmi.Value{
		"gmInput": wmi.Uint64(1 | 100<<8),
	})
	fmt.Printf("  rgbKb        : %s\n", raw(client, path, "GetGamingRgbKb", 1))
}

// bufferFor builds a backlight buffer with the level in the wanted byte.
func bufferFor(level int, position int) []byte {
	data := make([]byte, 16)
	data[position] = byte(level)
	data[9] = 1
	return data
}

// writeBuffer sends a backlight buffer.
func writeBuffer(client *wmi.Client, path string, data []byte) {
	_, err := client.Call(hardware.ClassGaming, "SetGamingKBBacklight", path, map[string]wmi.Value{
		"gmInput": wmi.Bytes(data),
	})
	if err != nil {
		fmt.Printf("  write failed: %v\n", err)
	}
}

// writeColour sends a colour for one zone.
func writeColour(client *wmi.Client, path string, red, green, blue uint8, zone uint64) {
	input := uint64(blue)<<24 | uint64(green)<<16 | uint64(red)<<8 | zone
	_, err := client.Call(hardware.ClassGaming, "SetGamingLEDColor", path, map[string]wmi.Value{
		"gmInput": wmi.Uint64(input),
	})
	if err != nil {
		fmt.Printf("  colour %d failed: %v\n", zone, err)
	}
}

// writeBehaviour sends an effect with its speed and direction.
func writeBehaviour(client *wmi.Client, path string, effect, speed, direction uint64) {
	input := effect<<16 | speed<<8 | direction
	_, err := client.Call(hardware.ClassGaming, "SetGamingLEDBehavior", path, map[string]wmi.Value{
		"gmInput": wmi.Uint64(input),
	})
	if err != nil {
		fmt.Printf("  behaviour failed: %v\n", err)
	}
}

// raw invokes a getter that answers with a number.
func raw(client *wmi.Client, path, method string, query uint32) string {
	outputs, err := client.Call(hardware.ClassGaming, method, path, map[string]wmi.Value{
		"gmInput": wmi.Uint32(query),
	})
	if err != nil {
		return "error: " + err.Error()
	}
	parts := make([]string, 0, 3)
	for _, name := range []string{"gmOutput", "gmReturn"} {
		if value, ok := outputs[name]; ok {
			parts = append(parts, name+"="+value.String())
		}
	}
	return strings.Join(parts, " ")
}

// buffer invokes a getter that answers with a byte array.
func buffer(client *wmi.Client, path, method string, query uint32) string {
	outputs, err := client.Call(hardware.ClassGaming, method, path, map[string]wmi.Value{
		"gmInput": wmi.Uint32(query),
	})
	if err != nil {
		return "error: " + err.Error()
	}
	parts := make([]string, 0, 3)
	for _, name := range []string{"gmOutput", "gmReturn"} {
		value, ok := outputs[name]
		if !ok {
			continue
		}
		if value.Kind == wmi.KindBytes {
			parts = append(parts, name+"="+hex.EncodeToString(value.Bytes))
		} else {
			parts = append(parts, name+"="+value.String())
		}
	}
	return strings.Join(parts, " ")
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
