//go:build windows

package nvidia

import (
	"testing"
	"unsafe"
)

func pointerOf(addr uintptr) unsafe.Pointer {
	if addr == 0 {
		return nil
	}
	return *(*unsafe.Pointer)(unsafe.Pointer(&addr))
}

func TestReadOfACard(t *testing.T) {

	lib := &library{
		init: func(arguments ...uintptr) (uintptr, bool) { return 0, true },
		deviceCount: func(arguments ...uintptr) (uintptr, bool) {
			count := (*uint32)(pointerOf(arguments[0]))
			*count = 1
			return 0, true
		},
		deviceHandle: func(arguments ...uintptr) (uintptr, bool) {
			handle := (*uintptr)(pointerOf(arguments[1]))
			*handle = 0x1234
			return 0, true
		},
		deviceUtilization: func(arguments ...uintptr) (uintptr, bool) {
			load := (*utilization)(pointerOf(arguments[1]))
			load.GPU = 35
			load.Memory = 12
			return 0, true
		},
		deviceClock: func(arguments ...uintptr) (uintptr, bool) {
			clock := (*uint32)(pointerOf(arguments[2]))
			*clock = 210
			return 0, true
		},
		deviceMaxClock: func(arguments ...uintptr) (uintptr, bool) {
			clock := (*uint32)(pointerOf(arguments[2]))
			*clock = 3105
			return 0, true
		},
		deviceName: func(arguments ...uintptr) (uintptr, bool) {
			buffer := (*[96]byte)(pointerOf(arguments[1]))
			copy(buffer[:], "NVIDIA GeForce RTX 4060")
			return 0, true
		},
		deviceMemory: func(arguments ...uintptr) (uintptr, bool) {
			info := (*memory)(pointerOf(arguments[1]))
			info.Total = 8 << 30
			info.Used = 2 << 30
			info.Free = 6 << 30
			return 0, true
		},
		devicePState: func(arguments ...uintptr) (uintptr, bool) {
			state := (*uint32)(pointerOf(arguments[1]))
			*state = 2
			return 0, true
		},
	}

	telemetry := read(lib, 0x1234)

	if telemetry.Name != "NVIDIA GeForce RTX 4060" {
		t.Errorf("the card is called %q, want the name of the card", telemetry.Name)
	}
	if telemetry.UsagePercent != 35 {
		t.Errorf("the card reports the load %d percent, want 35", telemetry.UsagePercent)
	}
	if telemetry.MemoryUsagePercent != 12 {
		t.Errorf("the card reports the load of the memory %d percent, want 12", telemetry.MemoryUsagePercent)
	}
	if telemetry.GraphicsClockMHz != 210 {
		t.Errorf("the card runs at %d megahertz, want 210", telemetry.GraphicsClockMHz)
	}
	if telemetry.MaxGraphicsClockMHz != 3105 {
		t.Errorf("the card reaches %d megahertz, want 3105", telemetry.MaxGraphicsClockMHz)
	}
	if telemetry.MemoryTotalBytes != 8<<30 {
		t.Errorf("the card holds %d bytes, want eight gigabytes", telemetry.MemoryTotalBytes)
	}
	if telemetry.MemoryUsedBytes != 2<<30 {
		t.Errorf("the card uses %d bytes, want two gigabytes", telemetry.MemoryUsedBytes)
	}
	if telemetry.PState != 2 {
		t.Errorf("the card is in the state %d, want 2", telemetry.PState)
	}
}

func TestReadOfACardThatDoesNotReportEverything(t *testing.T) {

	lib := &library{
		deviceName: nil,
	}

	telemetry := read(lib, 0)
	if telemetry.Name != "" {
		t.Errorf("the card is called %q, want an empty text", telemetry.Name)
	}
	if telemetry.UsagePercent != 0 {
		t.Errorf("the card reports the load %d percent, want zero", telemetry.UsagePercent)
	}

	if telemetry.PState != -1 {
		t.Errorf("the card is in the state %d, want minus one", telemetry.PState)
	}
}

func TestReadOfACardWhoseMemoryCallIsRefused(t *testing.T) {

	lib := &library{
		deviceMemory: func(arguments ...uintptr) (uintptr, bool) {
			return 0x80000001, true
		},
		deviceMemoryV2: func(arguments ...uintptr) (uintptr, bool) {
			info := (*memoryV2)(pointerOf(arguments[1]))
			info.Total = 4 << 30
			info.Used = 1 << 30
			return 0, true
		},
	}

	telemetry := read(lib, 0)
	if telemetry.MemoryTotalBytes != 4<<30 {
		t.Errorf("the card holds %d bytes, want four gigabytes", telemetry.MemoryTotalBytes)
	}
	if telemetry.MemoryUsedBytes != 1<<30 {
		t.Errorf("the card uses %d bytes, want one gigabyte", telemetry.MemoryUsedBytes)
	}
}

func TestReadNameOfACardWithoutAName(t *testing.T) {

	lib := &library{
		deviceName: func(arguments ...uintptr) (uintptr, bool) {
			return 0, true
		},
	}

	name, ok := readName(lib, 0)
	if !ok {
		t.Fatal("the name was not read")
	}
	if name != "" {
		t.Errorf("the card is called %q, want an empty text", name)
	}
}

func TestReadNameOfACardThatIsRefused(t *testing.T) {
	lib := &library{
		deviceName: func(arguments ...uintptr) (uintptr, bool) {
			return 1, true
		},
	}

	if _, ok := readName(lib, 0); ok {
		t.Error("the name of a card that refused the call was read")
	}
}

func TestTheCallOfAMissingEntryPoint(t *testing.T) {

	if _, ok := call(nil); ok {
		t.Error("an entry point that is not there was called")
	}
	result, ok := call(func(arguments ...uintptr) (uintptr, bool) { return 42, true })
	if !ok {
		t.Error("an entry point that is there was not called")
	}
	if result != 42 {
		t.Errorf("the entry point answered %d, want 42", result)
	}
}
