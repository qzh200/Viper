package hardware

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

// rsmb is the four character code for the raw SMBIOS table.
const rsmb = 0x52534D42

// systemInformationType is the SMBIOS structure type that carries the
// manufacturer, the product name and the family.
const systemInformationType = 1

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
func SystemIdentity() (model, family, product string) {
	identity, err := readIdentity()
	if err != nil {
		return "", "", ""
	}
	return identity.Product, identity.Family, identity.Product
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

// getSystemFirmwareTable reads a raw firmware table.
//
// The function is not wrapped by golang.org/x/sys/windows, so it is
// declared here. It is part of kernel32 and only reads the tables the
// firmware already exposes, it never writes anything.
var (
	kernel32                   = windows.NewLazySystemDLL("kernel32.dll")
	procGetSystemFirmwareTable = kernel32.NewProc("GetSystemFirmwareTable")
)

// getSystemFirmwareTable returns the size of the table when buffer is nil,
// and fills the buffer otherwise.
func getSystemFirmwareTable(provider uint32, tableID uint32, buffer []byte) uint32 {
	var pointer uintptr
	if len(buffer) > 0 {
		pointer = uintptr(unsafe.Pointer(&buffer[0]))
	}
	r, _, _ := procGetSystemFirmwareTable.Call(
		uintptr(provider),
		uintptr(tableID),
		pointer,
		uintptr(len(buffer)),
	)
	return uint32(r)
}

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
