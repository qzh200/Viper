//go:build tools

// Command sideeffect shows what changes on the Windows side when a
// miscellaneous setting is switched.
//
// The remaining unidentified settings are switches, so whatever they touch
// outside of the firmware identifies them.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"viper/internal/hardware"
	"viper/internal/wmi"
)

func main() {
	acpi, path := connectACPI()

	for _, id := range []uint64{0x01, 0x06, 0x08} {
		fmt.Printf("\n=== switching setting 0x%02X ===\n", id)
		before := snapshot()

		if _, err := acpi.Call(hardware.ClassGaming, "SetGamingMiscSetting", path, map[string]wmi.Value{
			"gmInput": wmi.Uint64(1<<8 | id),
		}); err != nil {
			fmt.Println("  failed:", err)
			continue
		}

		after := snapshot()
		for key, value := range after {
			if before[key] != value {
				fmt.Printf("  CHANGED %s\n    before: %s\n    after : %s\n", key, before[key], value)
			}
		}

		// Put the original value back.
		original, _ := readMisc(acpi, path, id)
		acpi.Call(hardware.ClassGaming, "SetGamingMiscSetting", path, map[string]wmi.Value{
			"gmInput": wmi.Uint64(original>>8<<8 | id),
		})
	}
}

// snapshot collects the state of the Windows side.
func snapshot() map[string]string {
	out := make(map[string]string)
	out["powercfg"] = run("powercfg", "/query")
	out["stickykeys"] = run("powershell", "-NoProfile", "-Command",
		"(Get-ItemProperty 'HKCU:\\Control Panel\\Accessibility\\StickyKeys' -ErrorAction SilentlyContinue).Flags")
	out["powercfg-battery"] = run("powercfg", "/query", "SCHEME_CURRENT", "SUB-BATTERY")
	out["powercfg-video"] = run("powercfg", "/query", "SCHEME_CURRENT", "SUB-VIDEO")
	return out
}

// run executes a command and returns its output.
func run(name string, args ...string) string {
	out, err := exec.Command(name, args...).CombinedOutput()
	if err != nil {
		return "error: " + err.Error()
	}
	return strings.TrimSpace(string(out))
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
	base := uint64(10)
	if strings.HasPrefix(text, "0x") || strings.HasPrefix(text, "0X") {
		text = text[2:]
		base = 16
	}
	return parseUint(text, base)
}

// parseUint parses a number.
func parseUint(text string, base uint64) (uint64, error) {
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
			return 0, fmt.Errorf("not a number: %q", text)
		}
		out = out*base + digit
	}
	return out, nil
}

// connectACPI opens the ACPI namespace.
func connectACPI() (*wmi.Client, string) {
	client, err := wmi.Connect(wmi.NamespaceACPI)
	if err != nil {
		fmt.Println("wmi:", err)
		exit(1)
	}
	paths, err := client.Instances(hardware.ClassGaming)
	if err != nil || len(paths) == 0 {
		fmt.Println("no instance:", err)
		exit(1)
	}
	return client, paths[0]
}

// exit stops the program.
func exit(code int) { os.Exit(code) }
