package hardware

import (
	"fmt"
)

// rsmb is the four character code for the raw SMBIOS table.
const rsmb = 0x52534D42

// systemInformationType is the SMBIOS structure type that carries the
// manufacturer, the product name and the family.
const systemInformationType = 1

// processorInformationType is the SMBIOS structure type that carries the
// clocks of the processor.
const processorInformationType = 4

// memoryDeviceType is the SMBIOS structure type that carries the speed of
// the memory modules.
const memoryDeviceType = 17

// Identity describes the machine as reported by SMBIOS.
type Identity struct {
	// Manufacturer is the company that built the machine.
	Manufacturer string
	// Product is the product name.
	Product string
	// Family is the product family, for example "Predator".
	Family string
	// Version is the product version.
	Version string
	// SerialNumber identifies the unit.
	SerialNumber string
}

// SystemIdentity reads the system information from the firmware tables.
//
// This is the only reliable way to tell a Predator from a Nitro: both
// expose the same WMI provider, but only the Predator offers the turbo
// performance profile.
//
// 三个返回值分别是机器的型号（固件里的产品名，例如 "Predator PHN16-71"）、
// 产品系列（例如 "Predator"）和厂商（例如 "Acer"）。以前第三个返回值也是
// 产品名，等于把型号说两遍，厂商从来没有被返回过。
func SystemIdentity() (model, family, manufacturer string) {
	identity, err := readIdentity()
	if err != nil {
		return "", "", ""
	}
	return identity.Product, identity.Family, identity.Manufacturer
}

// SystemInfo reads the full system information block.
func SystemInfo() (Identity, error) {
	return readIdentity()
}

// IsPredatorModel reports whether the machine belongs to the Predator
// series.
func IsPredatorModel(model, family string) bool {
	return containsFold(model, "predator") || containsFold(family, "predator")
}

// IsPredator reports whether this machine is a Predator.
func IsPredator() bool {
	identity, err := readIdentity()
	if err != nil {
		return false
	}
	return IsPredatorModel(identity.Product, identity.Family)
}

// getSystemFirmwareTable is declared in the files of the platforms, it
// reads a raw firmware table.
//
// The function is not wrapped by golang.org/x/sys/windows, so it is
// declared in firmware_windows.go. It is part of kernel32 and only reads the
// tables the firmware already exposes, it never writes anything.

// readIdentity parses the SMBIOS table looking for the system information
// structure.
func readIdentity() (Identity, error) {
	identity := Identity{
		Manufacturer: "Acer",
		Product:      "Acer Gaming Laptop",
	}

	size := getSystemFirmwareTable(rsmb, 0, nil)
	if size == 0 {
		return identity, fmt.Errorf("hardware: the firmware table is not available")
	}

	buffer := make([]byte, size)
	if getSystemFirmwareTable(rsmb, 0, buffer) == 0 {
		return identity, fmt.Errorf("hardware: the firmware table could not be read")
	}

	// The raw SMBIOS data starts with an eight byte header, the structures
	// follow behind it.
	if len(buffer) < 8 {
		return identity, fmt.Errorf("hardware: the firmware table is truncated")
	}

	found, ok := parseIdentity(buffer[8:])
	if !ok {
		return identity, fmt.Errorf("hardware: no system information structure found")
	}
	return found, nil
}

// parseIdentity walks the structures of a raw SMBIOS table and returns the
// system information structure.
func parseIdentity(table []byte) (Identity, bool) {
	identity := Identity{
		Manufacturer: "Acer",
		Product:      "Acer Gaming Laptop",
	}

	for offset := 0; offset+4 <= len(table); {
		structType := table[offset]
		length := int(table[offset+1])
		if length < 4 || offset+length > len(table) {
			break
		}

		if structType == systemInformationType && length >= 8 {
			strings := collectStrings(table[offset+length:])
			identity.Manufacturer = pick(strings, table[offset+4])
			identity.Product = pick(strings, table[offset+5])
			identity.Version = pick(strings, table[offset+6])
			identity.SerialNumber = pick(strings, table[offset+7])
			if length >= 0x1B {
				identity.Family = pick(strings, table[offset+0x1A])
			}
			return identity, true
		}

		offset = nextStructure(table, offset, length)
	}

	return identity, false
}

// ProcessorSpeed describes the clocks of the processor.
type ProcessorSpeed struct {
	// MaxSpeedMHz is the highest clock the processor reaches.
	MaxSpeedMHz int
	// CurrentSpeedMHz is the clock the processor runs at.
	CurrentSpeedMHz int
}

// ProcessorClocks reads the clocks of the processor from the firmware.
//
// The firmware table carries the speeds of the processor, the highest one
// is not stored anywhere else in a way that Windows keeps up to date.
func ProcessorClocks() (ProcessorSpeed, error) {
	var speed ProcessorSpeed

	size := getSystemFirmwareTable(rsmb, 0, nil)
	if size == 0 {
		return speed, fmt.Errorf("hardware: the firmware table is not available")
	}

	buffer := make([]byte, size)
	if getSystemFirmwareTable(rsmb, 0, buffer) == 0 {
		return speed, fmt.Errorf("hardware: the firmware table could not be read")
	}
	if len(buffer) < 8 {
		return speed, fmt.Errorf("hardware: the firmware table is truncated")
	}

	table := buffer[8:]
	for offset := 0; offset+4 <= len(table); {
		structType := table[offset]
		length := int(table[offset+1])
		if length < 4 || offset+length > len(table) {
			break
		}

		if structType == processorInformationType && length >= 0x18 {
			speed.CurrentSpeedMHz = int(table[offset+0x16]) | int(table[offset+0x17])<<8
			maxSpeed := int(table[offset+0x10]) | int(table[offset+0x11])<<8
			if maxSpeed == 0xFFFF && length >= 0x32 {
				// The speed does not fit into the small field, the wide
				// one behind it carries it.
				wide := uint32(table[offset+0x2E]) |
					uint32(table[offset+0x2F])<<8 |
					uint32(table[offset+0x30])<<16 |
					uint32(table[offset+0x31])<<24
				maxSpeed = int(wide & 0xFFFF)
			}
			speed.MaxSpeedMHz = maxSpeed
			return speed, nil
		}

		offset = nextStructure(table, offset, length)
	}

	return speed, fmt.Errorf("hardware: no processor information found")
}

// MemoryFrequency reads the clock of the memory in megahertz.
//
// The firmware table carries the speed of the memory modules, the first
// module of the machine is enough.
func MemoryFrequency() (int, error) {
	size := getSystemFirmwareTable(rsmb, 0, nil)
	if size == 0 {
		return 0, fmt.Errorf("hardware: the firmware table is not available")
	}

	buffer := make([]byte, size)
	if getSystemFirmwareTable(rsmb, 0, buffer) == 0 {
		return 0, fmt.Errorf("hardware: the firmware table could not be read")
	}
	if len(buffer) < 8 {
		return 0, fmt.Errorf("hardware: the firmware table is truncated")
	}

	table := buffer[8:]
	for offset := 0; offset+4 <= len(table); {
		structType := table[offset]
		length := int(table[offset+1])
		if length < 4 || offset+length > len(table) {
			break
		}

		// The structure of a memory device carries the speed. The place
		// of the field depends on the version of the standard, both are
		// read.
		if structType == memoryDeviceType && length >= 0x17 {
			speed := int(table[offset+0x15]) | int(table[offset+0x16])<<8
			if speed == 0 && length >= 0x1D {
				speed = int(table[offset+0x1B]) | int(table[offset+0x1C])<<8
			}
			if speed > 0 {
				return speed, nil
			}
		}

		offset = nextStructure(table, offset, length)
	}

	return 0, fmt.Errorf("hardware: no memory device found")
}

// nextStructure advances past the current structure and its string table.
func nextStructure(table []byte, offset, length int) int {
	// The string table follows the formatted area and ends with a double
	// null byte.
	pos := offset + length
	for pos < len(table) {
		if table[pos] == 0 {
			pos++
			if pos < len(table) && table[pos] == 0 {
				return pos + 1
			}
			continue
		}
		pos++
	}
	return len(table)
}

// collectStrings reads the string table that follows a structure.
func collectStrings(table []byte) []string {
	var out []string
	start := 0
	for i := 0; i < len(table); i++ {
		if table[i] != 0 {
			continue
		}
		if i > start {
			out = append(out, string(table[start:i]))
		}
		start = i + 1
		if start < len(table) && table[start] == 0 {
			break
		}
	}
	return out
}

// pick returns the n-th string of the string table, or an empty string.
func pick(strings []string, index byte) string {
	if index == 0 || int(index) > len(strings) {
		return ""
	}
	return strings[index-1]
}

// containsFold reports whether haystack contains needle, ignoring case.
func containsFold(haystack, needle string) bool {
	return len(needle) > 0 && len(haystack) >= len(needle) &&
		indexFold(haystack, needle) >= 0
}

// indexFold finds needle inside haystack, ignoring case.
func indexFold(haystack, needle string) int {
	lowerHaystack := make([]byte, len(haystack))
	for i := 0; i < len(haystack); i++ {
		lowerHaystack[i] = lower(haystack[i])
	}
	lowerNeedle := make([]byte, len(needle))
	for i := 0; i < len(needle); i++ {
		lowerNeedle[i] = lower(needle[i])
	}
	for i := 0; i+len(lowerNeedle) <= len(lowerHaystack); i++ {
		match := true
		for j := range lowerNeedle {
			if lowerHaystack[i+j] != lowerNeedle[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}

// lower converts an ASCII letter to lower case.
func lower(c byte) byte {
	if c >= 'A' && c <= 'Z' {
		return c + ('a' - 'A')
	}
	return c
}
