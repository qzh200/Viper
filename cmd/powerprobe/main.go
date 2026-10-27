//go:build tools

// Command powerprobe asks the kernel about the clocks of the processor.
//
// The information of the power manager holds the highest clock the
// processor reaches and the clock it runs at.
package main

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

func main() {
	kernel32 := windows.NewLazySystemDLL("powrprof.dll")
	proc := kernel32.NewProc("CallNtPowerInformation")

	// ProcessorInformation is the level eleven.
	const processorInformation = 11

	// The structure of one processor.
	type processorPower struct {
		Number           uint32
		MaxMHz           uint32
		CurrentMHz       uint32
		MHzLimit         uint32
		MaxIdleState     uint32
		CurrentIdleState uint32
	}

	buffer := make([]byte, 4096)
	status, _, _ := proc.Call(
		uintptr(processorInformation),
		0,
		0,
		uintptr(unsafe.Pointer(&buffer[0])),
		uintptr(len(buffer)),
	)
	fmt.Printf("status = 0x%08X\n", uint32(status))

	count := uint32(len(buffer)) / 24
	for i := uint32(0); i < count && i < 4; i++ {
		info := (*processorPower)(unsafe.Pointer(&buffer[i*24]))
		fmt.Printf("  processor %d: max %d MHz, current %d MHz, limit %d MHz\n",
			info.Number, info.MaxMHz, info.CurrentMHz, info.MHzLimit)
	}
}
