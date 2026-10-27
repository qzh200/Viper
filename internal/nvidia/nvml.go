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
	"sync"
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

// entry is an entry point of the management library.
//
// The type of the entry point is a function and not a pointer to the
// procedure of the library, so that a test can put a stand for the
// driver in place of the real one. A machine without the driver never
// reaches the code that reads the card, so those branches would otherwise
// stay untested.
//
// The second result tells whether the entry point is there at all. Not
// every driver exports every entry point, and a missing one has to be
// told apart from one that failed.
type entry func(arguments ...uintptr) (result uintptr, present bool)

// library holds the entry points of the management library.
//
// The name of the card, the performance state and the version of the
// driver are optional, not every driver exports them.
type library struct {
	handle syscall.Handle

	init                entry
	shutdown            entry
	deviceCount         entry
	deviceHandle        entry
	deviceUtilization   entry
	deviceClock         entry
	deviceMaxClock      entry
	deviceName          entry
	deviceMemory        entry
	deviceMemoryV2      entry
	devicePState        entry
	systemDriverVersion entry
}

// wrap turns an entry point of the library into a call.
//
// Not every driver exports every entry point, so a missing one answers
// that it is not there, and the caller leaves the value alone.
func wrap(dll *syscall.LazyDLL, name string) entry {
	proc := dll.NewProc(name)
	return func(arguments ...uintptr) (uintptr, bool) {
		if err := proc.Find(); err != nil {
			return 0, false
		}
		result, _, _ := proc.Call(arguments...)
		return result, true
	}
}

// call runs an entry point of the library.
//
// The result is false when the entry point is missing, the caller then
// leaves the value alone.
func call(proc entry, arguments ...uintptr) (uintptr, bool) {
	if proc == nil {
		return 0, false
	}
	return proc(arguments...)
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
//
// The service answers every connection in its own goroutine, so the state
// is shared by all of them. It is guarded by a lock, because a reading
// that arrives while the library is being opened would otherwise see a
// half filled table and dereference a nil pointer.
var (
	mutex     sync.Mutex
	loaded    bool
	loadError error
	current   *library
)

// load opens the management library of the driver.
//
// The library is opened once and kept open, because opening it again for
// every reading would slow the dashboard down.
func load() (*library, error) {
	mutex.Lock()
	defer mutex.Unlock()

	if loaded {
		return current, loadError
	}

	handle, err := syscall.LoadLibrary("nvml.dll")
	if err != nil {
		loadError = fmt.Errorf("nvidia: the management library of the driver is not available: %w", err)
		current = nil
		loaded = true
		return nil, loadError
	}

	dll := syscall.NewLazyDLL("nvml.dll")
	current = &library{
		handle:              handle,
		init:                wrap(dll, "nvmlInit_v2"),
		shutdown:            wrap(dll, "nvmlShutdown"),
		deviceCount:         wrap(dll, "nvmlDeviceGetCount_v2"),
		deviceHandle:        wrap(dll, "nvmlDeviceGetHandleByIndex_v2"),
		deviceUtilization:   wrap(dll, "nvmlDeviceGetUtilizationRates"),
		deviceClock:         wrap(dll, "nvmlDeviceGetClockInfo"),
		deviceMaxClock:      wrap(dll, "nvmlDeviceGetMaxClockInfo"),
		deviceName:          wrap(dll, "nvmlDeviceGetName"),
		deviceMemory:        wrap(dll, "nvmlDeviceGetMemoryInfo"),
		deviceMemoryV2:      wrap(dll, "nvmlDeviceGetMemoryInfo_v2"),
		devicePState:        wrap(dll, "nvmlDeviceGetPState"),
		systemDriverVersion: wrap(dll, "nvmlSystemGetDriverVersion"),
	}
	loadError = nil
	loaded = true
	return current, nil
}

// Available reports whether the driver of an NVIDIA card is present.
func Available() bool {
	_, err := load()
	return err == nil
}

// Close releases the management library.
//
// The library is opened again by the next reading, so that the program
// survives the driver being installed while it runs.
func Close() {
	mutex.Lock()
	defer mutex.Unlock()

	if current == nil || current.handle == 0 {
		return
	}
	current.shutdown()
	syscall.FreeLibrary(current.handle)
	current = nil
	loadError = nil
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

	result, _ := lib.init()
	if int32(result) != 0 {
		return nil, fmt.Errorf("nvidia: the driver refused the access, code %d", int32(result))
	}

	var count uint32
	result, _ = lib.deviceCount(uintptr(unsafe.Pointer(&count)))
	if int32(result) != 0 {
		return nil, fmt.Errorf("nvidia: the cards cannot be counted, code %d", int32(result))
	}

	out := make([]Telemetry, 0, count)
	for index := uint32(0); index < count; index++ {
		var handle uintptr
		result, _ = lib.deviceHandle(
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
