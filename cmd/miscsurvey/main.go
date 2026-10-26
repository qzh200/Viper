//go:build tools

// Command miscsurvey maps the miscellaneous settings of the gaming
// provider.
//
// The known identifiers are eleven for the operation mode, nine for the
// discrete graphics and ten for the processor overclock, so the remaining
// ones are likely to hold the battery charging and the screen settings.
// Every identifier is read, then written and read again, which tells the
// writable ones apart from the read only ones.
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

	fmt.Println("=== reading every miscellaneous setting ===")
	values := make(map[uint64]uint64)
	for id := uint64(0); id <= 255; id++ {
		raw, err := readRaw(client, path, id)
		if err != nil {
			continue
		}
		if raw == 0 {
			continue
		}
		values[id] = raw
	}

	ids := make([]uint64, 0, len(values))
	for id := range values {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })

	for _, id := range ids {
		fmt.Printf("  id 0x%02X = 0x%X (%d)\n", id, values[id], values[id])
	}

	fmt.Println()
	fmt.Println("=== writing and reading back ===")
	for _, id := range ids {
		for _, value := range []uint64{1, 2} {
			before := values[id]
			if _, err := client.Call(hardware.ClassGaming, "SetGamingMiscSetting", path, map[string]wmi.Value{
				"gmInput": wmi.Uint64(value<<8 | id),
			}); err != nil {
				continue
			}
			after, err := readRaw(client, path, id)
			if err != nil || after == before {
				continue
			}
			fmt.Printf("  id 0x%02X: write %d -> 0x%X (was 0x%X)\n", id, value, after, before)
			// Restore the original value.
			client.Call(hardware.ClassGaming, "SetGamingMiscSetting", path, map[string]wmi.Value{
				"gmInput": wmi.Uint64(before>>8<<8 | id),
			})
			break
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
	var out uint64
	for _, c := range text {
		var digit uint64
		switch {
		case c >= '0' && c <= '9':
			digit = uint64(c - '0')
		case c >= 'a' && c <= 'f':
			digit = uint64(c-'a') + 10
		case c >= 'A' && c <= 'F':
			digit = uint64(c-'A') + 10
		default:
			return 0, fmt.Errorf("not a number: %q", value.String())
		}
		out = out*uint64(base) + digit
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
