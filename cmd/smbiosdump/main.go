// Command smbiosdump prints the raw structures of the firmware table.
//
// The fields of the structures are read by hand, so the bytes are shown
// next to the numbers that are taken from them.
//
// 表的结构由硬件层读出来，这个工具只负责把它摊开看。
package main

import (
	"fmt"
	"io"
	"os"

	"viper/internal/hardware"
)

// stdout is where the tool writes. 用例把它换成缓冲区。
var stdout io.Writer = os.Stdout

func main() {
	table, err := hardware.RawFirmwareTable()
	if err != nil {
		fmt.Fprintln(stdout, "读取固件表失败:", err)
		os.Exit(1)
	}

	show(table)
}

// show prints the structures of the firmware table.
func show(table []byte) {
	for _, structure := range hardware.Structures(table) {
		switch structure.Type {
		case 4:
			fmt.Fprintf(stdout, "=== processor information, length %d ===\n", len(structure.Data))
			dump(structure.Data)
			printProcessorFields(structure.Data)
		case 17:
			fmt.Fprintf(stdout, "=== memory device, length %d ===\n", len(structure.Data))
			dump(structure.Data)
			printMemoryFields(structure.Data)
		}
	}
}

// dump shows the bytes of a structure.
func dump(data []byte) {
	for i := 0; i < len(data); i += 16 {
		end := i + 16
		if end > len(data) {
			end = len(data)
		}
		fmt.Fprintf(stdout, "  0x%02X:", i)
		for j := i; j < end; j++ {
			fmt.Fprintf(stdout, " %02X", data[j])
		}
		fmt.Fprintln(stdout)
	}
}

// printProcessorFields shows the numbers that are taken from a processor
// structure.
func printProcessorFields(data []byte) {
	fmt.Fprintf(stdout, "  max speed   0x10: %d\n", hardware.Word(data, 0x10))
	fmt.Fprintf(stdout, "  current spd 0x12: %d\n", hardware.Word(data, 0x12))
	fmt.Fprintf(stdout, "  core count  0x14: %d\n", hardware.Word(data, 0x14))
	fmt.Fprintf(stdout, "  thread cnt  0x18: %d\n", hardware.Word(data, 0x18))
}

// printMemoryFields shows the numbers that are taken from a memory
// structure.
func printMemoryFields(data []byte) {
	fmt.Fprintf(stdout, "  size        0x10: %d\n", hardware.Word(data, 0x10))
	fmt.Fprintf(stdout, "  speed       0x1B: %d\n", hardware.Word(data, 0x1B))
	fmt.Fprintf(stdout, "  speed alt   0x15: %d\n", hardware.Word(data, 0x15))
}
