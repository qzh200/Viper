//go:build windows

// Command modeprobe compares the state of the machine across the
// operation mode encodings that the provider accepts.
//
// The provider keeps five values: zero, one, four, five and six. Two and
// three are clamped back to balanced. This probe looks at the surrounding
// profile readings to tell the five apart.
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

	for _, value := range []uint64{0, 1, 4, 5, 6} {
		if _, err := client.Call(hardware.ClassGaming, "SetGamingMiscSetting", path, map[string]wmi.Value{
			"gmInput": wmi.Uint64(value<<8 | 0x0B),
		}); err != nil {
			fmt.Printf("value %d -> %v\n", value, err)
			continue
		}

		mode, _ := read(client, path, "GetGamingMiscSetting", wmi.Uint32(0x0B))
		profile, _ := read(client, path, "GetGamingProfile", wmi.Uint32(0))
		profileSetting, _ := read(client, path, "GetGamingProfileSetting", wmi.Uint32(0))
		fanTable, _ := read(client, path, "GetGamingFanTable", wmi.Uint32(0))
		profile1, _ := read(client, path, "GetGamingProfile", wmi.Uint32(1))

		fmt.Printf("value %d -> mode=0x%X profile=0x%X profileSetting=0x%X fanTable=0x%X profile1=0x%X\n",
			value, mode, profile, profileSetting, fanTable, profile1)
	}

	fmt.Println()
	fmt.Println("=== restore balanced ===")
	if err := backend.SetPowerMode(hardware.PowerBalanced); err != nil {
		fmt.Println("  failed:", err)
	} else {
		mode, _ := backend.PowerMode()
		fmt.Println("  mode is now", mode)
	}
}

// read invokes a method and returns the raw numeric output.
func read(client *wmi.Client, path, method string, input wmi.Value) (uint64, error) {
	outputs, err := client.Call(hardware.ClassGaming, method, path, map[string]wmi.Value{
		"gmInput": input,
	})
	if err != nil {
		return 0, err
	}
	value, ok := outputs["gmOutput"]
	if !ok {
		return 0, fmt.Errorf("no gmOutput")
	}
	text := strings.TrimSpace(value.String())
	base := 10
	if strings.HasPrefix(text, "0x") || strings.HasPrefix(text, "0X") {
		text = text[2:]
		base = 16
	}
	return strconv.ParseUint(text, base, 64)
}
