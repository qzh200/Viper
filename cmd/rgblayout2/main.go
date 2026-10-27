//go:build tools

// Command rgblayout2 pins down the layout of the keyboard backlight
// buffer and the order in which the brightness reaches the keyboard.
//
// The provider reads the buffer back with GetGamingKBBacklight, so the
// layout is found by writing a buffer and reading it again.
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
	client, path, action := connect()

	restore(client, path, 100)

	fmt.Println("=== baseline ===")
	fmt.Printf("  kbBacklight(1) : %s\n", buffer(client, path, "GetGamingKBBacklight", 1))
	fmt.Printf("  rgbKb(1)       : %s\n", raw(client, path, "GetGamingRgbKb", 1))
	fmt.Printf("  action read    : %s\n", actionRead(client, action))

	// The buffer that the provider reports for the keyboard is
	// 00 00 64 00 00 64 00 00 03 00 ..., so the level sits in byte two
	// and in byte five. The candidates below follow that shape.
	candidates := []struct {
		name string
		data []byte
	}{
		{"byte2 + byte5 = 42, byte8 = 3", []byte{0, 0, 42, 0, 0, 42, 0, 0, 3, 0, 0, 0, 0, 0, 0, 0}},
		{"byte2 = 42 only", []byte{0, 0, 42, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}},
		{"byte5 = 42 only", []byte{0, 0, 0, 0, 0, 42, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}},
		{"byte2 + byte5 = 42, byte9 = 1", []byte{0, 0, 42, 0, 0, 42, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0}},
		{"byte2 + byte5 = 42, byte8 = 3, byte9 = 1", []byte{0, 0, 42, 0, 0, 42, 0, 0, 3, 1, 0, 0, 0, 0, 0, 0}},
	}

	for _, candidate := range candidates {
		fmt.Printf("--- %s ---\n", candidate.name)
		writeBuffer(client, path, candidate.data)
		fmt.Printf("  kbBacklight(1) : %s\n", buffer(client, path, "GetGamingKBBacklight", 1))
		fmt.Printf("  rgbKb(1)       : %s\n", raw(client, path, "GetGamingRgbKb", 1))
		fmt.Printf("  action read    : %s\n", actionRead(client, action))
	}

	fmt.Println("\n=== order of the three writes ===")
	restore(client, path, 100)
	fmt.Printf("  start          : %s\n", raw(client, path, "GetGamingRgbKb", 1))
	actionWrite(client, action, 42)
	fmt.Printf("  after action   : %s\n", raw(client, path, "GetGamingRgbKb", 1))
	writeBuffer(client, path, []byte{0, 0, 42, 0, 0, 42, 0, 0, 3, 0, 0, 0, 0, 0, 0, 0})
	fmt.Printf("  after buffer   : %s\n", raw(client, path, "GetGamingRgbKb", 1))
	writeRgbKb(client, path, 42)
	fmt.Printf("  after rgbKb    : %s\n", raw(client, path, "GetGamingRgbKb", 1))
	fmt.Printf("  kbBacklight(1) : %s\n", buffer(client, path, "GetGamingKBBacklight", 1))

	fmt.Println("\n=== restore ===")
	restore(client, path, 100)
	fmt.Printf("  rgbKb(1)       : %s\n", raw(client, path, "GetGamingRgbKb", 1))
	fmt.Printf("  kbBacklight(1) : %s\n", buffer(client, path, "GetGamingKBBacklight", 1))
}

// restore puts the brightness back.
func restore(client *wmi.Client, path string, level int) {
	writeRgbKb(client, path, uint64(level))
}

// writeRgbKb writes the level with the gaming class.
func writeRgbKb(client *wmi.Client, path string, level uint64) {
	_, err := client.Call(hardware.ClassGaming, "SetGamingRgbKb", path, map[string]wmi.Value{
		"gmInput": wmi.Uint64(1 | level<<8),
	})
	if err != nil {
		fmt.Println("  rgbKb write failed:", err)
	}
}

// writeBuffer sends a backlight buffer.
func writeBuffer(client *wmi.Client, path string, data []byte) {
	_, err := client.Call(hardware.ClassGaming, "SetGamingKBBacklight", path, map[string]wmi.Value{
		"gmInput": wmi.Bytes(data),
	})
	if err != nil {
		fmt.Println("  buffer write failed:", err)
	}
}

// actionWrite writes the level with the action class.
func actionWrite(client *wmi.Client, path string, level uint64) {
	input := uint64(2) | 132<<8 | 0x80000 | level<<32
	_, err := client.Call(hardware.ClassAction, "SetFunction", path, map[string]wmi.Value{
		"uiInput": wmi.Uint64(input),
	})
	if err != nil {
		fmt.Println("  action write failed:", err)
	}
}

// actionRead reads the level with the action class.
func actionRead(client *wmi.Client, path string) string {
	query := uint64(1) | 132<<8 | 0x80000
	outputs, err := client.Call(hardware.ClassAction, "GetFunction", path, map[string]wmi.Value{
		"uiInput": wmi.Uint32(uint32(query)),
	})
	if err != nil {
		return "error: " + err.Error()
	}
	value, ok := outputs["uiOutput"]
	if !ok {
		return "no output"
	}
	return value.String()
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
func connect() (*wmi.Client, string, string) {
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
	paths2, err := client.Instances(hardware.ClassAction)
	if err == nil && len(paths2) > 0 {
		return client, paths[0], paths2[0]
	}
	return client, paths[0], paths[0]
}
