//go:build windows

// Package nvidia reads the telemetry of the graphics card from the
// driver of the vendor.
//
// The driver offers two sets of calls. The interface of the driver itself
// exports only a single function and hides everything else behind a table
// of numbers, the management library exports the calls by name and is much
// easier to use. Viper takes the management library.
//
// The library belongs to the driver, so it is loaded when it is needed and
// a machine without an NVIDIA card simply has no telemetry.
package nvidia

import (
	"fmt"
	"syscall"
	"unsafe"
)

// The clock kinds the driver knows.
const (
	// clockGraphics is the graphics clock.
	clockGraphics = 0
	// clockMemory is the memory clock.
	clockMemory = 2
)

// utilization is the load of the card.
type utilization struct {
	GPU    uint32
	Memory uint32
}

// memory is the memory of the card.
//
// The first version of the call carries three numbers, the second one a
// version field in front of them. Not every driver knows the second one.
type memory struct {
	Total uint64
	Free  uint64
	Used  uint64
}

// memoryV2 is the structure of the second version of the call.
type memoryV2 struct {
	Version uint64
	Total   uint64
	Free    uint64
	Used    uint64
}

// library holds the entry points of the management library.
//
// The name of the card, the performance state and the version of the
// driver are optional, not every driver exports them.
type library struct {
	handle syscall.Handle

	init                *syscall.LazyProc
	shutdown            *syscall.LazyProc
	deviceCount         *syscall.LazyProc
	deviceHandle        *syscall.LazyProc
	deviceUtilization   *syscall.LazyProc
	deviceClock         *syscall.LazyProc
	deviceMaxClock      *syscall.LazyProc
	deviceName          *syscall.LazyProc
	deviceMemory        *syscall.LazyProc
	deviceMemoryV2      *syscall.LazyProc
	devicePState        *syscall.LazyProc
	systemDriverVersion *syscall.LazyProc
}

// Telemetry is a snapshot of the graphics card.
type Telemetry struct {
	// Name is the name of the card.
	Name string
	// UsagePercent is the load of the graphics processor in percent.
	UsagePercent int
	// MemoryUsagePercent is the load of the memory in percent.
	MemoryUsagePercent int
	// GraphicsClockMHz is the current graphics clock.
	GraphicsClockMHz int
	// MemoryClockMHz is the current memory clock.
	MemoryClockMHz int
	// MaxGraphicsClockMHz is the highest graphics clock the card reaches.
	MaxGraphicsClockMHz int
	// MemoryUsedBytes is the memory in use by the graphics processor.
	MemoryUsedBytes uint64
	// MemoryTotalBytes is the memory the card has.
	MemoryTotalBytes uint64
	// PState is the performance state of the card. The lower the number,
	// the faster the card works. It is minus one when the driver does not
	// report it.
	PState int
	// Idle reports whether the card is doing nothing.
	Idle bool
}

// Idle reports whether the card sits idle.
//
// The driver does not always offer the performance state, so the load of
// the graphics processor is used to tell.
func (t Telemetry) IsIdle() bool {
	return t.UsagePercent == 0
}

// The state of the library.
var (
	loaded    bool
	loadError error
	entry     *library
)

// load opens the management library of the driver.
//
// The library is opened once and kept open, because opening it again for
// every reading would slow the dashboard down.
func load() (*library, error) {
	if loaded {
		return entry, loadError
	}
	loaded = true

	handle, err := syscall.LoadLibrary("nvml.dll")
	if err != nil {
		loadError = fmt.Errorf("nvidia: the management library of the driver is not available: %w", err)
		return nil, loadError
	}

	dll := syscall.NewLazyDLL("nvml.dll")
	entry = &library{
		handle:              handle,
		init:                dll.NewProc("nvmlInit_v2"),
		shutdown:            dll.NewProc("nvmlShutdown"),
		deviceCount:         dll.NewProc("nvmlDeviceGetCount_v2"),
		deviceHandle:        dll.NewProc("nvmlDeviceGetHandleByIndex_v2"),
		deviceUtilization:   dll.NewProc("nvmlDeviceGetUtilizationRates"),
		deviceClock:         dll.NewProc("nvmlDeviceGetClockInfo"),
		deviceMaxClock:      dll.NewProc("nvmlDeviceGetMaxClockInfo"),
		deviceName:          dll.NewProc("nvmlDeviceGetName"),
		deviceMemory:        dll.NewProc("nvmlDeviceGetMemoryInfo"),
		deviceMemoryV2:      dll.NewProc("nvmlDeviceGetMemoryInfo_v2"),
		devicePState:        dll.NewProc("nvmlDeviceGetPState"),
		systemDriverVersion: dll.NewProc("nvmlSystemGetDriverVersion"),
	}
	return entry, nil
}

// Available reports whether the driver of an NVIDIA card is present.
func Available() bool {
	_, err := load()
	return err == nil
}

// Close releases the management library.
func Close() {
	if entry == nil || entry.handle == 0 {
		return
	}
	entry.shutdown.Call()
	syscall.FreeLibrary(entry.handle)
	entry = nil
	loaded = false
}

// Read reads the state of every graphics card of the machine.
//
// A machine without an NVIDIA card answers with an empty list, the caller
// does not have to tell the two cases apart.
func Read() ([]Telemetry, error) {
	lib, err := load()
	if err != nil {
		return nil, err
	}

	result, _, _ := lib.init.Call()
	if int32(result) != 0 {
		return nil, fmt.Errorf("nvidia: the driver refused the access, code %d", int32(result))
	}

	var count uint32
	result, _, _ = lib.deviceCount.Call(uintptr(unsafe.Pointer(&count)))
	if int32(result) != 0 {
		return nil, fmt.Errorf("nvidia: the cards cannot be counted, code %d", int32(result))
	}

	out := make([]Telemetry, 0, count)
	for index := uint32(0); index < count; index++ {
		var handle uintptr
		result, _, _ = lib.deviceHandle.Call(
			uintptr(index),
			uintptr(unsafe.Pointer(&handle)),
		)
		if int32(result) != 0 {
			continue
		}
		out = append(out, read(lib, handle))
	}
	return out, nil
}

// call runs an entry point that the driver may not export.
//
// The result is false when the entry point is missing, the caller then
// leaves the value alone.
func call(proc *syscall.LazyProc, arguments ...uintptr) (uintptr, bool) {
	if proc == nil {
		return 0, false
	}
	if err := proc.Find(); err != nil {
		return 0, false
	}
	result, _, _ := proc.Call(arguments...)
	return result, true
}

// read reads the state of one card.
func read(lib *library, handle uintptr) Telemetry {
	telemetry := Telemetry{PState: -1}

	if name, ok := readName(lib, handle); ok {
		telemetry.Name = name
	}

	var load utilization
	if result, ok := call(lib.deviceUtilization,
		handle,
		uintptr(unsafe.Pointer(&load)),
	); ok && int32(result) == 0 {
		telemetry.UsagePercent = int(load.GPU)
		telemetry.MemoryUsagePercent = int(load.Memory)
	}

	var clock uint32
	if result, ok := call(lib.deviceClock,
		handle,
		uintptr(clockGraphics),
		uintptr(unsafe.Pointer(&clock)),
	); ok && int32(result) == 0 {
		telemetry.GraphicsClockMHz = int(clock)
	}

	if result, ok := call(lib.deviceClock,
		handle,
		uintptr(clockMemory),
		uintptr(unsafe.Pointer(&clock)),
	); ok && int32(result) == 0 {
		telemetry.MemoryClockMHz = int(clock)
	}

	if result, ok := call(lib.deviceMaxClock,
		handle,
		uintptr(clockGraphics),
		uintptr(unsafe.Pointer(&clock)),
	); ok && int32(result) == 0 {
		telemetry.MaxGraphicsClockMHz = int(clock)
	}

	// The driver of this machine knows the first version of the call, the
	// second one is refused with the code for an unsupported feature.
	var info memory
	if result, ok := call(lib.deviceMemory,
		handle,
		uintptr(unsafe.Pointer(&info)),
	); ok && int32(result) == 0 {
		telemetry.MemoryUsedBytes = info.Used
		telemetry.MemoryTotalBytes = info.Total
	} else {
		var infoV2 memoryV2
		if result, ok := call(lib.deviceMemoryV2,
			handle,
			uintptr(unsafe.Pointer(&infoV2)),
		); ok && int32(result) == 0 {
			telemetry.MemoryUsedBytes = infoV2.Used
			telemetry.MemoryTotalBytes = infoV2.Total
		}
	}

	var state uint32
	if result, ok := call(lib.devicePState,
		handle,
		uintptr(unsafe.Pointer(&state)),
	); ok && int32(result) == 0 {
		telemetry.PState = int(state)
	}

	return telemetry
}

// readName reads the name of a card.
func readName(lib *library, handle uintptr) (string, bool) {
	buffer := make([]byte, 96)
	result, ok := call(lib.deviceName,
		handle,
		uintptr(unsafe.Pointer(&buffer[0])),
		uintptr(len(buffer)),
	)
	if !ok || int32(result) != 0 {
		return "", false
	}
	for i, b := range buffer {
		if b == 0 {
			return string(buffer[:i]), true
		}
	}
	return string(buffer), true
}

// DriverVersion reads the version of the driver.
func DriverVersion() (string, error) {
	lib, err := load()
	if err != nil {
		return "", err
	}
	buffer := make([]byte, 96)
	result, ok := call(lib.systemDriverVersion,
		uintptr(unsafe.Pointer(&buffer[0])),
		uintptr(len(buffer)),
	)
	if !ok {
		return "", fmt.Errorf("nvidia: the driver does not report its version")
	}
	if int32(result) != 0 {
		return "", fmt.Errorf("nvidia: the version of the driver cannot be read, code %d", int32(result))
	}
	for i, b := range buffer {
		if b == 0 {
			return string(buffer[:i]), nil
		}
	}
	return string(buffer), nil
}
