//go:build tools

// Command nvmlmem looks for the call that reports the memory of the
// graphics card.
//
// The two versions of the call take a different structure, so both are
// tried and the return codes are shown.
package main

import (
	"fmt"
	"syscall"
	"unsafe"
)

func main() {
	dll := syscall.NewLazyDLL("nvml.dll")

	if result, _, _ := dll.NewProc("nvmlInit_v2").Call(); int32(result) != 0 {
		fmt.Println("the driver refused the access:", int32(result))
		return
	}

	var count uint32
	dll.NewProc("nvmlDeviceGetCount_v2").Call(uintptr(unsafe.Pointer(&count)))
	fmt.Println("cards:", count)

	var handle uintptr
	dll.NewProc("nvmlDeviceGetHandleByIndex_v2").Call(0, uintptr(unsafe.Pointer(&handle)))

	// The first version: three numbers.
	type memoryV1 struct {
		Total uint64
		Free  uint64
		Used  uint64
	}
	// The second version: a version number in front of the numbers.
	type memoryV2 struct {
		Version uint64
		Total   uint64
		Free    uint64
		Used    uint64
	}

	for _, name := range []string{"nvmlDeviceGetMemoryInfo", "nvmlDeviceGetMemoryInfo_v2"} {
		proc := dll.NewProc(name)
		if err := proc.Find(); err != nil {
			fmt.Printf("%s is not exported\n", name)
			continue
		}
		v1 := memoryV1{}
		result, _, _ := proc.Call(handle, uintptr(unsafe.Pointer(&v1)))
		fmt.Printf("%s -> code %d  total=%d free=%d used=%d\n",
			name, int32(result), v1.Total, v1.Free, v1.Used)

		v2 := memoryV2{}
		result, _, _ = proc.Call(handle, uintptr(unsafe.Pointer(&v2)))
		fmt.Printf("%s (v2 layout) -> code %d  version=%d total=%d free=%d used=%d\n",
			name, int32(result), v2.Version, v2.Total, v2.Free, v2.Used)
	}

	// The name of the card and the performance state.
	name := make([]byte, 96)
	result, _, _ := dll.NewProc("nvmlDeviceGetName").Call(handle, uintptr(unsafe.Pointer(&name[0])), uintptr(len(name)))
	fmt.Printf("name -> code %d  %s\n", int32(result), string(name[:indexOfZero(name)]))

	if proc := dll.NewProc("nvmlDeviceGetPState"); proc.Find() == nil {
		var state uint32
		result, _, _ = proc.Call(handle, uintptr(unsafe.Pointer(&state)))
		fmt.Printf("pstate -> code %d  %d\n", int32(result), state)
	} else {
		fmt.Println("nvmlDeviceGetPState is not exported")
	}

	dll.NewProc("nvmlShutdown").Call()
}

// indexOfZero finds the end of a text.
func indexOfZero(data []byte) int {
	for i, b := range data {
		if b == 0 {
			return i
		}
	}
	return len(data)
}
