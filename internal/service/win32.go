//go:build windows

package service

import (
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows/registry"
)

// The kernel library of Windows.
var (
	kernel32Library        = syscall.NewLazyDLL("kernel32.dll")
	procGetSystemTimes     = kernel32Library.NewProc("GetSystemTimes")
	procGlobalMemoryStatus = kernel32Library.NewProc("GlobalMemoryStatusEx")
)

// fileTime is the time of the system, in hundred nanoseconds.
type fileTime struct {
	LowDateTime  uint32
	HighDateTime uint32
}

// systemTimes carries the times of the processor.
type systemTimes struct {
	Idle   fileTime
	Kernel fileTime
	User   fileTime
}

// memoryStatus is the memory of the machine.
type memoryStatus struct {
	Length                   uint32
	MemoryLoad               uint32
	TotalPhysical            uint64
	AvailablePhysical        uint64
	TotalPageFile            uint64
	AvailablePageFile        uint64
	TotalVirtual             uint64
	AvailableVirtual         uint64
	AvailableExtendedVirtual uint64
}

// processorTimes holds the times of the last reading.
var processorTimes struct {
	previous     systemTimes
	havePrevious bool
}

// primeCPUUsage takes the first reading of the times of the processor.
//
// The load is the share of the time that the processor was not idle since
// the last reading, so the first answer needs a reading to compare with.
func primeCPUUsage() {
	cpuUsage()
}

// cpuUsage returns the load of the processor in percent.
//
// The load is the share of the time that the processor was not idle since
// the last call, so the first call has nothing to compare with.
func cpuUsage() float64 {
	var times systemTimes
	result, _, _ := procGetSystemTimes.Call(
		uintptr(unsafe.Pointer(&times.Idle)),
		uintptr(unsafe.Pointer(&times.Kernel)),
		uintptr(unsafe.Pointer(&times.User)),
	)
	if result == 0 {
		return 0
	}

	current := systemTimes{
		Idle:   times.Idle,
		Kernel: times.Kernel,
		User:   times.User,
	}
	if !processorTimes.havePrevious {
		processorTimes.previous = current
		processorTimes.havePrevious = true
		return 0
	}

	previous := processorTimes.previous
	processorTimes.previous = current

	idle := current.Idle.seconds() - previous.Idle.seconds()
	kernel := current.Kernel.seconds() - previous.Kernel.seconds()
	user := current.User.seconds() - previous.User.seconds()

	total := kernel + user
	if total <= 0 {
		return 0
	}
	// The idle time is part of the kernel time.
	busy := total - idle
	if busy < 0 {
		busy = 0
	}
	return busy / total * 100
}

// seconds turns the time of the system into seconds.
func (f fileTime) seconds() float64 {
	ticks := uint64(f.HighDateTime)<<32 | uint64(f.LowDateTime)
	return float64(ticks) / 1e7
}

// memoryInfo reads the memory of the machine.
//
// The first result is the total amount in bytes, the second the share that
// is in use in percent.
func memoryInfo() (uint64, float64) {
	status := memoryStatus{Length: uint32(unsafe.Sizeof(memoryStatus{}))}
	result, _, _ := procGlobalMemoryStatus.Call(uintptr(unsafe.Pointer(&status)))
	if result == 0 {
		return 0, 0
	}

	used := status.TotalPhysical - status.AvailablePhysical
	share := 0.0
	if status.TotalPhysical > 0 {
		share = float64(used) / float64(status.TotalPhysical) * 100
	}
	return status.TotalPhysical, share
}

// processorClock reads the clock of the processor in megahertz.
//
// The value comes from the description of the hardware, Windows keeps it up
// to date.
func processorClock() int {
	key, err := registry.OpenKey(registry.LOCAL_MACHINE,
		`HARDWARE\DESCRIPTION\System\CentralProcessor\0`, registry.QUERY_VALUE)
	if err != nil {
		return 0
	}
	defer key.Close()

	value, _, err := key.GetIntegerValue("~MHz")
	if err != nil {
		return 0
	}
	return int(value)
}

// processorName reads the name of the processor.
func processorName() string {
	key, err := registry.OpenKey(registry.LOCAL_MACHINE,
		`HARDWARE\DESCRIPTION\System\CentralProcessor\0`, registry.QUERY_VALUE)
	if err != nil {
		return ""
	}
	defer key.Close()

	name, _, err := key.GetStringValue("ProcessorNameString")
	if err != nil {
		return ""
	}
	return name
}

// processorCount returns the number of the logical processors.
func processorCount() int {
	key, err := registry.OpenKey(registry.LOCAL_MACHINE,
		`HARDWARE\DESCRIPTION\System\CentralProcessor`, registry.ENUMERATE_SUB_KEYS)
	if err != nil {
		return 0
	}
	defer key.Close()

	names, err := key.ReadSubKeyNames(-1)
	if err != nil {
		return 0
	}
	return len(names)
}
