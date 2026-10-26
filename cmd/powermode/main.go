//go:build windows

// Command powermode investigates how the performance profile is encoded.
//
// It scans every miscellaneous setting and then tries a range of encodings
// for the operation mode, restoring the original profile afterwards.
package main

import (
	"fmt"
	"os"
	"strconv"
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

	fmt.Println("=== current operation mode (setting 0x0B) ===")
	raw, _ := rawValue(client, path, "GetGamingMiscSetting", wmi.Uint32(0x0B))
	fmt.Printf("  raw = %s\n", raw)

	fmt.Println()
	fmt.Println("=== GetGamingProfile / GetGamingProfileSetting ===")
	for _, method := range []string{"GetGamingProfile", "GetGamingProfileSetting"} {
		for _, query := range []uint32{0, 1, 2, 4, 0x0B} {
			value, err := rawValue(client, path, method, wmi.Uint32(query))
			fmt.Printf("  %s(%d) = %s (err=%v)\n", method, query, value, err)
		}
	}

	fmt.Println()
	fmt.Println("=== scan of every miscellaneous setting ===")
	for id := 0; id < 256; id++ {
		value, err := rawValue(client, path, "GetGamingMiscSetting", wmi.Uint32(uint32(id)))
		if err != nil {
			continue
		}
		if n, ok := parse(value); ok && n != 0 {
			fmt.Printf("  id 0x%02X = %s\n", id, value)
		}
	}

	fmt.Println()
	fmt.Println("=== try operation mode encodings ===")
	original, _ := backend.PowerMode()
	fmt.Printf("  original mode: %s\n", original)
	for value := 0; value <= 255; value++ {
		if _, err := client.Call(hardware.ClassGaming, "SetGamingMiscSetting", path, map[string]wmi.Value{
			"gmInput": wmi.Uint64(uint64(value)<<8 | 0x0B),
		}); err != nil {
			continue
		}
		current, err := backend.PowerMode()
		if err != nil {
			continue
		}
		// Only interesting encodings are printed: the ones the provider
		// actually keeps.
		if current != hardware.PowerBalanced {
			fmt.Printf("  value %3d -> mode %s\n", value, current)
		}
	}

	fmt.Println()
	fmt.Println("=== restore ===")
	if err := backend.SetPowerMode(original); err != nil {
		fmt.Println("  restore failed:", err)
	} else {
		current, _ := backend.PowerMode()
		fmt.Printf("  mode is now %s\n", current)
	}
}

// rawValue invokes a method and renders the raw output value.
func rawValue(client *wmi.Client, path, method string, input wmi.Value) (string, error) {
	outputs, err := client.Call(hardware.ClassGaming, method, path, map[string]wmi.Value{
		"gmInput": input,
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
		text := value.String()
		if n, ok := parse(text); ok {
			text = fmt.Sprintf("%s (0x%X)", text, n)
		}
		parts = append(parts, name+"="+text)
	}
	if len(parts) == 0 {
		return "", fmt.Errorf("no output parameters")
	}
	return strings.Join(parts, " "), nil
}

// parse converts a rendered value into a number.
func parse(text string) (uint64, bool) {
	text = strings.TrimSpace(text)
	if text == "" {
		return 0, false
	}
	base := 10
	if strings.HasPrefix(text, "0x") || strings.HasPrefix(text, "0X") {
		text = text[2:]
		base = 16
	}
	n, err := strconv.ParseUint(text, base, 64)
	if err != nil {
		return 0, false
	}
	return n, true
}
