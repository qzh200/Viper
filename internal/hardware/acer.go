// Package hardware drives the Acer ACPI WMI provider.
//
// The provider exposes two classes: AcerGamingFunction, which offers the
// named getters and setters of the gaming hardware, and APGeAction, the
// generic ACPI PNP0C14 action class used by the keyboard and CoolBoost.
//
// Every method signature here was read from the live MOF, which matters
// because the input types differ per method and some methods take no input
// at all.
package hardware

import (
	"fmt"
	"strconv"
	"strings"

	"viper/internal/wmi"
)

// Class names of the ACPI WMI provider.
const (
	// ClassGaming exposes the Acer gaming hardware functions.
	ClassGaming = "AcerGamingFunction"

	// ClassAction exposes the generic ACPI PNP0C14 action functions.
	ClassAction = "APGeAction"
)

// Sensor categories of the GetGamingSysInfo method. The query is encoded
// as category<<8 | index.
const (
	sysInfoCPUTemp    = 0x0101
	sysInfoCPUFanRPM  = 0x0201
	sysInfoSystemTemp = 0x0301
	sysInfoGPUFanRPM  = 0x0601
	sysInfoGPUTemp    = 0x0A01
)

// Sensors is a snapshot of the machine temperatures and fan speeds.
type Sensors struct {
	CPUTempC    int
	CPUFanRPM   int
	GPUTempC    int
	GPUFanRPM   int
	SystemTempC int
}

// Backend talks to the Acer ACPI WMI provider.
type Backend struct {
	client     *wmi.Client
	gamingPath string
	actionPath string
}

// NewBackend connects to the ACPI namespace and resolves the instances that
// carry the gaming functions.
//
// Enumerating the instances requires administrative rights.
func NewBackend() (*Backend, error) {
	client, err := wmi.Connect(wmi.NamespaceACPI)
	if err != nil {
		return nil, err
	}

	backend := &Backend{client: client}

	gamingPaths, err := client.Instances(ClassGaming)
	if err != nil {
		client.Close()
		return nil, err
	}
	if len(gamingPaths) == 0 {
		client.Close()
		return nil, fmt.Errorf("hardware: no %s instance found", ClassGaming)
	}
	backend.gamingPath = gamingPaths[0]

	// APGeAction usually shares the ACPI device with AcerGamingFunction,
	// but the class has to be resolved separately.
	actionPaths, err := client.Instances(ClassAction)
	if err != nil || len(actionPaths) == 0 {
		// Fall back to the instance of the gaming class, which is what the
		// provider exposes on most machines.
		backend.actionPath = backend.gamingPath
	} else {
		backend.actionPath = actionPaths[0]
	}

	return backend, nil
}

// Instance returns the resolved object path of the gaming functions.
func (b *Backend) Instance() string { return b.gamingPath }

// Close releases the WMI connection.
func (b *Backend) Close() { b.client.Close() }

// callGaming invokes a method of the AcerGamingFunction class.
func (b *Backend) callGaming(method string, args map[string]wmi.Value) (map[string]wmi.Value, error) {
	return b.client.Call(ClassGaming, method, b.gamingPath, args)
}

// callAction invokes a method of the APGeAction class.
func (b *Backend) callAction(method string, args map[string]wmi.Value) (map[string]wmi.Value, error) {
	return b.client.Call(ClassAction, method, b.actionPath, args)
}

// Sensors reads the current temperatures and fan speeds.
func (b *Backend) Sensors() (Sensors, error) {
	var out Sensors

	cpuTemp, err := b.sysInfo(sysInfoCPUTemp)
	if err != nil {
		return out, err
	}
	out.CPUTempC = cpuTemp

	cpuFan, err := b.sysInfo(sysInfoCPUFanRPM)
	if err != nil {
		return out, err
	}
	out.CPUFanRPM = cpuFan

	sysTemp, err := b.sysInfo(sysInfoSystemTemp)
	if err != nil {
		return out, err
	}
	out.SystemTempC = sysTemp

	gpuFan, err := b.sysInfo(sysInfoGPUFanRPM)
	if err != nil {
		return out, err
	}
	out.GPUFanRPM = gpuFan

	gpuTemp, err := b.sysInfo(sysInfoGPUTemp)
	if err != nil {
		return out, err
	}
	out.GPUTempC = gpuTemp

	return out, nil
}

// sysInfo reads one value through GetGamingSysInfo.
//
// The provider packs the reading into the upper bits of the result, the
// low byte carrying the status.
func (b *Backend) sysInfo(query uint32) (int, error) {
	outputs, err := b.callGaming("GetGamingSysInfo", map[string]wmi.Value{
		"gmInput": wmi.Uint32(query),
	})
	if err != nil {
		return 0, err
	}
	value, ok := outputs["gmOutput"]
	if !ok {
		return 0, fmt.Errorf("hardware: GetGamingSysInfo returned no gmOutput")
	}

	raw, err := decodeUint(value)
	if err != nil {
		return 0, err
	}
	if raw>>8 != 0 {
		// Fan speeds use the full 16 bit field, temperatures the low byte.
		return int((raw >> 8) & 0xFFFF), nil
	}
	return int(raw & 0xFF), nil
}

// decodeUint extracts a numeric value from a WMI result.
//
// The Acer provider returns most readings as a hexadecimal string, so both
// representations have to be accepted.
func decodeUint(value wmi.Value) (uint64, error) {
	if value.Kind == wmi.KindString {
		text := strings.TrimSpace(value.Str)
		if text == "" {
			return 0, nil
		}
		base := 10
		if strings.HasPrefix(text, "0x") || strings.HasPrefix(text, "0X") {
			text = text[2:]
			base = 16
		}
		n, err := strconv.ParseUint(text, base, 64)
		if err != nil {
			return 0, fmt.Errorf("hardware: cannot parse %q: %w", value.Str, err)
		}
		return n, nil
	}
	return value.UInt64(), nil
}
