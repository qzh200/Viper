// Command smbiosdump prints the raw structures of the firmware table.
//
// The fields of the structures are read by hand, so the bytes are shown
// next to the numbers that are taken from them.
package main

import (
	"fmt"
	"os"
	"unsafe"

	"golang.org/x/sys/windows"
)

const rsmb = 0x52534D42

func main() {
	kernel32 := windows.NewLazySystemDLL("kernel32.dll")
	proc := kernel32.NewProc("GetSystemFirmwareTable")

	size, _, _ := proc.Call(uintptr(rsmb), 0, 0, 0)
	if size == 0 {
		fmt.Println("the table is not available")
		os.Exit(1)
	}

	buffer := make([]byte, size)
	proc.Call(uintptr(rsmb), 0, uintptr(unsafe.Pointer(&buffer[0])), uintptr(len(buffer)))

	table := buffer[8:]
	for offset := 0; offset+4 <= len(table); {
		structType := table[offset]
		length := int(table[offset+1])
		if length < 4 || offset+length > len(table) {
			break
		}

		switch structType {
		case 4:
			fmt.Printf("=== processor information, length %d ===\n", length)
			dump(table[offset : offset+length])
			printProcessorFields(table[offset : offset+length])
		case 17:
			fmt.Printf("=== memory device, length %d ===\n", length)
			dump(table[offset : offset+length])
			printMemoryFields(table[offset : offset+length])
		}

		pos := offset + length
		for pos < len(table) {
			if table[pos] == 0 {
				pos++
				if pos < len(table) && table[pos] == 0 {
					pos++
					break
				}
				continue
			}
			pos++
		}
		offset = pos
	}
}

// dump shows the bytes of a structure.
func dump(data []byte) {
	for i := 0; i < len(data); i += 16 {
		end := i + 16
		if end > len(data) {
			end = len(data)
		}
		fmt.Printf("  0x%02X:", i)
		for j := i; j < end; j++ {
			fmt.Printf(" %02X", data[j])
		}
		fmt.Println()
	}
}

// printProcessorFields shows the numbers that are taken from a processor
// structure.
func printProcessorFields(data []byte) {
	fmt.Printf("  max speed   0x10: %d\n", word(data, 0x10))
	fmt.Printf("  current spd 0x12: %d\n", word(data, 0x12))
	fmt.Printf("  core count  0x14: %d\n", word(data, 0x14))
	fmt.Printf("  thread cnt  0x18: %d\n", word(data, 0x18))
}

// printMemoryFields shows the numbers that are taken from a memory
// structure.
func printMemoryFields(data []byte) {
	fmt.Printf("  size        0x10: %d\n", word(data, 0x10))
	fmt.Printf("  speed       0x1B: %d\n", word(data, 0x1B))
	fmt.Printf("  speed alt   0x15: %d\n", word(data, 0x15))
}

// word reads a little endian word.
func word(data []byte, offset int) int {
	if offset+2 > len(data) {
		return -1
	}
	return int(data[offset]) | int(data[offset+1])<<8
}
