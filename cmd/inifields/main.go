//go:build tools

// Command inifields shows which fields of the lighting profile change when
// the parameters are written over WMI.
//
// The profile of the Acer lighting service is the only place where the
// active mode is kept, so it doubles as the state of the keyboard.
package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"viper/internal/hardware"
	"viper/internal/wmi"
)

const profilePath = `C:\ProgramData\OEM\AcerLightingService\LightingProfile\LightingProfile.ini`

func main() {
	client, path := connect()

	before := readSections()

	fmt.Println("=== write the parameters of the meteor mode ===")
	if _, err := client.Call(hardware.ClassGaming, "SetGamingLEDBehavior", path, map[string]wmi.Value{
		"gmInput": wmi.Uint64(6<<16 | 7<<8 | 1),
	}); err != nil {
		fmt.Println("  failed:", err)
	}
	time.Sleep(900 * time.Millisecond)

	after := readSections()

	fmt.Println()
	fmt.Println("=== fields that changed ===")
	for _, name := range sectionNames(before) {
		old := before[name]
		new := after[name]
		if sameFields(old, new) {
			continue
		}
		fmt.Printf("  [%s]\n", name)
		for _, key := range []string{"value", "flags", "brightness", "color_mode", "color", "direction", "speed"} {
			if pick(old, key) != pick(new, key) {
				fmt.Printf("      %-11s %q -> %q\n", key, pick(old, key), pick(new, key))
			}
		}
	}

	fmt.Println()
	fmt.Println("=== the profile as it is now ===")
	for _, name := range sectionNames(after) {
		fmt.Printf("  %-34s value=%-4s flags=%-4s brightness=%-4s color_mode=%-3s color=%-10s dir=%-3s speed=%s\n",
			name, pick(after[name], "value"), pick(after[name], "flags"), pick(after[name], "brightness"),
			pick(after[name], "color_mode"), pick(after[name], "color"),
			pick(after[name], "direction"), pick(after[name], "speed"))
	}
}

// sameFields reports whether two sections carry the same values.
func sameFields(a, b map[string]string) bool {
	for key, value := range a {
		if b[key] != value {
			return false
		}
	}
	for key, value := range b {
		if a[key] != value {
			return false
		}
	}
	return true
}

// sectionNames returns the section names of a profile in order.
func sectionNames(sections map[string]map[string]string) []string {
	var names []string
	for name := range sections {
		names = append(names, name)
	}
	// The device section comes first, the mode sections follow in file
	// order, which a plain map does not preserve, so the caller only uses
	// this for lookups.
	return names
}

// pick returns a value of a section.
func pick(section map[string]string, key string) string {
	return section[key]
}

// readSections parses the lighting profile into its sections.
func readSections() map[string]map[string]string {
	sections := make(map[string]map[string]string)
	text, err := os.ReadFile(profilePath)
	if err != nil {
		return sections
	}
	current := ""
	for _, line := range strings.Split(string(text), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			current = strings.Trim(line, "[]")
			sections[current] = make(map[string]string)
			continue
		}
		if current == "" {
			continue
		}
		if key, value, found := strings.Cut(line, "="); found {
			sections[current][strings.TrimSpace(key)] = strings.TrimSpace(value)
		}
	}
	return sections
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
