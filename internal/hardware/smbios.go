package hardware

import (
	"fmt"
)

// SMBIOS 固件表签名与类型常量。
const (
	rsmb                     = 0x52534D42 // 'RSMB' 提供者签名
	systemInformationType    = 1          // Type 1: 系统信息
	processorInformationType = 4          // Type 4: 处理器信息
	memoryDeviceType         = 17         // Type 17: 内存设备信息
)

// Identity 包含从 SMBIOS 提取的计算机硬件标识数据。
type Identity struct {
	// Manufacturer 制造商名称。
	Manufacturer string

	// Product 产品型号名称。
	Product string

	// Family 产品系列名称。
	Family string

	// Version 硬件/主板版本。
	Version string

	// SerialNumber 设备序列号。
	SerialNumber string
}

// SystemIdentity 返回简化的设备型号、产品家族与制造商。
func SystemIdentity() (model, family, manufacturer string) {
	identity, err := readIdentity()
	if err != nil {
		return "", "", ""
	}
	return identity.Product, identity.Family, identity.Manufacturer
}

// SystemInfo 从固件 SMBIOS 表中读取并解析系统身份信息。
func SystemInfo() (Identity, error) {
	return readIdentity()
}

// IsPredatorModel 根据型号与系列字符串判断是否属于 Acer Predator（掠夺者）系列。
func IsPredatorModel(model, family string) bool {
	return containsFold(model, "predator") || containsFold(family, "predator")
}

// IsPredator 检测当前机器是否属于 Acer Predator 系列。
func IsPredator() bool {
	identity, err := readIdentity()
	if err != nil {
		return false
	}
	return IsPredatorModel(identity.Product, identity.Family)
}

func readIdentity() (Identity, error) {
	identity := Identity{
		Manufacturer: "Acer",
		Product:      "Acer Gaming Laptop",
	}

	table, err := RawFirmwareTable()
	if err != nil {
		return identity, err
	}

	found, ok := parseIdentity(table)
	if !ok {
		return identity, fmt.Errorf("hardware: no system information structure found")
	}
	return found, nil
}

// Structure 表示解析后的单个 SMBIOS 结构体。
type Structure struct {
	// Type 结构体类型标识。
	Type byte

	// Data 格式化数据区域。
	Data []byte

	// Offset 在原始表中的字节偏移。
	Offset int

	// Strings 附加的以 null 结尾的字符串表。
	Strings []string
}

// Structures 解析 SMBIOS 原始字节数据，枚举出所有的 SMBIOS 结构单元。
func Structures(table []byte) []Structure {
	var out []Structure
	for offset := 0; offset+4 <= len(table); {
		structType := table[offset]
		length := int(table[offset+1])
		if length < 4 || offset+length > len(table) {
			break
		}

		out = append(out, Structure{
			Type:    structType,
			Data:    table[offset : offset+length],
			Offset:  offset,
			Strings: collectStrings(table[offset+length:]),
		})
		offset = nextStructure(table, offset, length)
	}
	return out
}

// RawFirmwareTable 读取未经解析的 SMBIOS 固件原始表字节数据。
func RawFirmwareTable() ([]byte, error) {
	size := getSystemFirmwareTable(rsmb, 0, nil)
	if size == 0 {
		return nil, fmt.Errorf("hardware: the firmware table is not available")
	}

	buffer := make([]byte, size)
	if getSystemFirmwareTable(rsmb, 0, buffer) == 0 {
		return nil, fmt.Errorf("hardware: the firmware table could not be read")
	}
	if len(buffer) < 8 {
		return nil, fmt.Errorf("hardware: the firmware table is truncated")
	}
	return buffer[8:], nil
}

func parseIdentity(table []byte) (Identity, bool) {
	identity := Identity{
		Manufacturer: "Acer",
		Product:      "Acer Gaming Laptop",
	}

	for _, structure := range Structures(table) {
		if structure.Type != systemInformationType || len(structure.Data) < 8 {
			continue
		}
		identity.Manufacturer = pick(structure.Strings, structure.Data[4])
		identity.Product = pick(structure.Strings, structure.Data[5])
		identity.Version = pick(structure.Strings, structure.Data[6])
		identity.SerialNumber = pick(structure.Strings, structure.Data[7])
		if len(structure.Data) >= 0x1B {
			identity.Family = pick(structure.Strings, structure.Data[0x1A])
		}
		return identity, true
	}

	return identity, false
}

// ProcessorSpeed 记录处理器主频信息。
type ProcessorSpeed struct {
	MaxSpeedMHz     int
	CurrentSpeedMHz int
}

// Word 从字节切片的指定偏移处读取 16 位小端无符号整数。
func Word(data []byte, offset int) int {
	if offset+2 > len(data) {
		return -1
	}
	return int(data[offset]) | int(data[offset+1])<<8
}

func nextStructure(table []byte, offset, length int) int {
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

func pick(strings []string, index byte) string {
	if index == 0 || int(index) > len(strings) {
		return ""
	}
	return strings[index-1]
}

func containsFold(haystack, needle string) bool {
	return len(needle) > 0 && len(haystack) >= len(needle) &&
		indexFold(haystack, needle) >= 0
}

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

func lower(c byte) byte {
	if c >= 'A' && c <= 'Z' {
		return c + ('a' - 'A')
	}
	return c
}
