//go:build tools

// Command writeprobe tries to change every miscellaneous setting and
// reports the ones that move.
//
// The discrete graphics switch and the overclock setting are left alone
// because changing them has a visible effect on the machine.
package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"viper/internal/hardware"
	"viper/internal/wmi"
)

// skipped are the settings that must not be touched.
var skipped = map[uint64]string{
	0x09: "discrete graphics",
	0x0A: "processor overclock",
}

func main() {
	client, path := connect()

	for id := uint64(0); id <= 0x20; id++ {
		if reason, ok := skipped[id]; ok {
			fmt.Printf("  id 0x%02X skipped (%s)\n", id, reason)
			continue
		}

		before, err := readRaw(client, path, id)
		if err != nil {
			continue
		}

		var changes []string
		for value := uint64(0); value <= 128; value++ {
			if _, err := client.Call(hardware.ClassGaming, "SetGamingMiscSetting", path, map[string]wmi.Value{
				"gmInput": wmi.Uint64(value<<8 | id),
			}); err != nil {
				continue
			}
			after, err := readRaw(client, path, id)
			if err != nil || after == before {
				continue
			}
			changes = append(changes, fmt.Sprintf("write %d -> 0x%X", value, after))
			// Put the original value back.
			client.Call(hardware.ClassGaming, "SetGamingMiscSetting", path, map[string]wmi.Value{
				"gmInput": wmi.Uint64(before>>8<<8 | id),
			})
			if len(changes) >= 4 {
				break
			}
		}

		if len(changes) > 0 {
			fmt.Printf("  id 0x%02X (now 0x%X): %s\n", id, before, strings.Join(changes, ", "))
		} else {
			fmt.Printf("  id 0x%02X (now 0x%X): no change\n", id, before)
		}
	}
}

// readRaw reads one miscellaneous setting.
func readRaw(client *wmi.Client, path string, id uint64) (uint64, error) {
	outputs, err := client.Call(hardware.ClassGaming, "GetGamingMiscSetting", path, map[string]wmi.Value{
		"gmInput": wmi.Uint32(uint32(id)),
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
	out, err := strconv.ParseUint(text, base, 64)
	if err != nil {
		return 0, err
	}
	return out, nil
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
