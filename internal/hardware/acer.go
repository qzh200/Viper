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
	sysInfoCPUFanRPM  = 0x0201
	sysInfoSystemTemp = 0x0301
	sysInfoGPUFanRPM  = 0x0601
	sysInfoCPUTemp    = 0x0101
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
	client   *wmi.Client
	instance string
}

// NewBackend connects to the ACPI namespace and resolves the instance that
// carries the gaming functions.
//
// Enumerating the instances requires administrative rights.
func NewBackend() (*Backend, error) {
	client, err := wmi.Connect(wmi.NamespaceACPI)
	if err != nil {
		return nil, err
	}

	paths, err := client.Instances(ClassGaming)
	if err != nil {
		client.Close()
		return nil, err
	}
	if len(paths) == 0 {
		client.Close()
		return nil, fmt.Errorf("hardware: no %s instance found", ClassGaming)
	}

	return &Backend{client: client, instance: paths[0]}, nil
}

// Instance returns the resolved object path of the gaming functions.
func (b *Backend) Instance() string { return b.instance }

// Close releases the WMI connection.
func (b *Backend) Close() { b.client.Close() }

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
	outputs, err := b.client.Call(ClassGaming, "GetGamingSysInfo", b.instance, map[string]wmi.Value{
		"gmInput": wmi.Uint32(query),
	})
	if err != nil {
		return 0, err
	}
	value, ok := outputs["gmOutput"]
	if !ok {
		return 0, fmt.Errorf("hardware: GetGamingSysInfo returned no gmOutput")
	}
	return decodeSysInfo(value), nil
}

// decodeSysInfo extracts the numeric reading from a GetGamingSysInfo result.
//
// The result is a string in the provider's own encoding, for example
// "0x0001" for the CPU temperature category.
func decodeSysInfo(value wmi.Value) int {
	text := strings.TrimSpace(value.String())
	if text == "" {
		return 0
	}

	var raw uint64
	if strings.HasPrefix(text, "0x") || strings.HasPrefix(text, "0X") {
		if n, err := parseUint(text[2:], 16); err == nil {
			raw = n
		}
	} else if n, err := parseUint(text, 10); err == nil {
		raw = n
	}

	switch {
	case raw>>8 != 0:
		// Fan speeds use the full 16 bit field, temperatures the low byte.
		return int((raw >> 8) & 0xFFFF)
	default:
		return int(raw & 0xFF)
	}
}

// parseUint parses an unsigned integer in the given base.
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
			return 0, fmt.Errorf("hardware: invalid digit %q", c)
		}
		out = out*base + digit
	}
	return out, nil
}
