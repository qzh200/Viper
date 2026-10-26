//go:build tools

// Command batteryprobe finds out which miscellaneous setting controls the
// battery charging.
//
// The battery state is read from the standard CIM namespace before and
// after every candidate setting is switched.
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
	acpi, path := connectACPI()
	cim := connectCIM()

	before := batteryState(cim)
	fmt.Println("=== battery before ===")
	printState(before)

	for _, id := range []uint64{0x01, 0x06, 0x08, 0x02} {
		fmt.Printf("\n=== switching setting 0x%02X ===\n", id)
		if _, err := acpi.Call(hardware.ClassGaming, "SetGamingMiscSetting", path, map[string]wmi.Value{
			"gmInput": wmi.Uint64(1<<8 | id),
		}); err != nil {
			fmt.Println("  failed:", err)
			continue
		}
		after := batteryState(cim)
		printState(after)
		for key, value := range after {
			if before[key] != value {
				fmt.Printf("  CHANGED %s: %q -> %q\n", key, before[key], value)
			}
		}
		// Put it back to the original value.
		original, _ := readMisc(acpi, path, id)
		acpi.Call(hardware.ClassGaming, "SetGamingMiscSetting", path, map[string]wmi.Value{
			"gmInput": wmi.Uint64(original>>8<<8 | id),
		})
	}
}

// batteryState reads every property of the battery.
func batteryState(client *wmi.Client) map[string]string {
	out := make(map[string]string)
	paths, err := client.Instances("Win32_Battery")
	if err != nil || len(paths) == 0 {
		out["error"] = "no battery"
		return out
	}
	for _, name := range []string{
		"BatteryStatus", "Status", "EstimatedChargeRemaining", "EstimatedRunTime",
		"DesignCapacity", "FullChargeCapacity", "DesignVoltage", "DeviceID",
		"Chemistry", "BatteryRechargeRate", "BatteryDischargeRate",
	} {
		if value, err := client.ReadProperty(paths[0], name); err == nil {
			out[name] = strconv.FormatUint(value, 10)
		}
	}
	return out
}

// printState shows the battery state.
func printState(state map[string]string) {
	for key, value := range state {
		fmt.Printf("  %-28s %s\n", key, value)
	}
}

// readMisc reads one miscellaneous setting.
func readMisc(client *wmi.Client, path string, id uint64) (uint64, error) {
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
	return strconv.ParseUint(text, base, 64)
}

// connectACPI opens the ACPI namespace.
func connectACPI() (*wmi.Client, string) {
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

// connectCIM opens the standard namespace.
func connectCIM() *wmi.Client {
	client, err := wmi.Connect(wmi.NamespaceCIM)
	if err != nil {
		fmt.Println("wmi cim:", err)
		os.Exit(1)
	}
	return client
}
