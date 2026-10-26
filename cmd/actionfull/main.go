//go:build tools

// Command actionfull scans the whole query space of the generic ACPI
// action class.
//
// The keyboard backlight and the CoolBoost state are read from this class,
// so the battery charging settings are likely to be there as well.
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
	client, path := connect()

	fmt.Println("=== reading the action class ===")
	for query := uint64(0); query <= 0x400; query++ {
		raw, err := readAction(client, path, query)
		if err != nil || raw == 0 {
			continue
		}
		// The meaningful answers are small numbers, the rest are the
		// error code of the class.
		if raw > 0x10000 {
			continue
		}
		fmt.Printf("  GetFunction(0x%03X) = %d (0x%X)\n", query, raw, raw)
	}
}

// readAction reads one value of the generic action class.
func readAction(client *wmi.Client, path string, query uint64) (uint64, error) {
	outputs, err := client.Call(hardware.ClassAction, "GetFunction", path, map[string]wmi.Value{
		"uiInput": wmi.Uint32(uint32(query)),
	})
	if err != nil {
		return 0, err
	}
	value, ok := outputs["uiOutput"]
	if !ok {
		return 0, fmt.Errorf("no uiOutput")
	}
	text := strings.TrimSpace(value.String())
	base := 10
	if strings.HasPrefix(text, "0x") || strings.HasPrefix(text, "0X") {
		text = text[2:]
		base = 16
	}
	out, err := strconv.ParseUint(text, base, 64)
	if err != nil {
		return 0, err
	}
	return out, nil
}

// connect opens the ACPI namespace and returns the action instance path.
func connect() (*wmi.Client, string) {
	client, err := wmi.Connect(wmi.NamespaceACPI)
	if err != nil {
		fmt.Println("wmi:", err)
		os.Exit(1)
	}
	paths, err := client.Instances(hardware.ClassAction)
	if err != nil || len(paths) == 0 {
		paths, err = client.Instances(hardware.ClassGaming)
		if err != nil || len(paths) == 0 {
			fmt.Println("no instance:", err)
			os.Exit(1)
		}
	}
	return client, paths[0]
}
