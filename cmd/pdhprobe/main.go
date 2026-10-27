//go:build tools

// Command pdhprobe reads the performance counters of Windows.
//
// The counters of the processor carry the clock of every core and the
// share of the highest clock that is used.
package main

import (
	"fmt"
	"os"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// The functions of the performance data helper.
var (
	pdh                   = windows.NewLazyDLL("pdh.dll")
	procOpenQuery         = pdh.NewProc("PdhOpenQuery")
	procAddEnglishCounter = pdh.NewProc("PdhAddEnglishCounter")
	procCollectQueryData  = pdh.NewProc("PdhCollectQueryData")
	procGetFormattedArray = pdh.NewProc("PdhGetFormattedCounterArray")
	procGetFormattedValue = pdh.NewProc("PdhGetFormattedCounterValue")
	procCloseQuery        = pdh.NewProc("PdhCloseQuery")
	procFreeCounterArray  = pdh.NewProc("PdhFreeMemory")
)

// The counter value is a structure with a status and a number.
type counterValue struct {
	Status uint32
	Value  uint64
}

// The array of counter values.
type counterValues struct {
	Count  uint32
	Values uintptr
}

func main() {
	var query uintptr
	status, _, _ := procOpenQuery.Call(0, 0, 0, uintptr(unsafe.Pointer(&query)))
	if status != 0 {
		fmt.Printf("PdhOpenQuery -> 0x%08X\n", status)
		os.Exit(1)
	}
	defer procCloseQuery.Call(query)

	paths := []string{
		`\Processor Information(*)\Processor Frequency`,
		`\Processor Information(_Total)\Processor Frequency`,
		`\Processor Information(_Total)\% Processor Performance`,
		`\Processor Information(_Total)\% of Maximum Frequency`,
	}

	for _, path := range paths {
		var counter uintptr
		status, _, _ := procAddEnglishCounter.Call(
			query,
			uintptr(unsafe.Pointer(&path)),
			0,
			uintptr(unsafe.Pointer(&counter)),
		)
		if status != 0 {
			fmt.Printf("%-56s -> 0x%08X\n", path, status)
			continue
		}
		fmt.Printf("%-56s -> counter %d\n", path, counter)

		procCollectQueryData.Call(query)
		time.Sleep(200 * time.Millisecond)

		var array counterValues
		status, _, _ = procGetFormattedArray.Call(
			counter,
			2, // PDH_FMT_NOCAP100
			1, // PDH_FMT_LARGE
			0,
			0,
			uintptr(unsafe.Pointer(&array)),
		)
		if status == 0 && array.Count > 0 {
			values := (*[256]counterValue)(unsafe.Pointer(array.Values))
			fmt.Printf("   %d values:", array.Count)
			for i := uint32(0); i < array.Count && i < 6; i++ {
				fmt.Printf(" %d", values[i].Value)
			}
			fmt.Println()
			procFreeCounterArray.Call(array.Values)
		}

		var single counterValue
		status, _, _ = procGetFormattedValue.Call(
			counter,
			2,
			1,
			0,
			uintptr(unsafe.Pointer(&single)),
		)
		if status == 0 {
			fmt.Printf("   single: %d (status %d)\n", single.Value, single.Status)
		}
	}
}
