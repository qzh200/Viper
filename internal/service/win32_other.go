//go:build !windows

package service

// The readings of the processor and of the memory come from the kernel of
// Windows. On every other system there is no such reading, so the service
// reports nothing. The names are the same on every system, so that the
// rest of the package does not have to know where it runs.

// primeCPUUsage takes the first reading of the times of the processor.
//
// There is nothing to read on this system.
func primeCPUUsage() {}

// cpuUsage returns the load of the processor in percent.
//
// There is nothing to read on this system.
func cpuUsage() float64 {
	return 0
}

// memoryInfo returns the used memory in bytes and the load in percent.
//
// There is nothing to read on this system.
func memoryInfo() (uint64, float64) {
	return 0, 0
}

// processorClock returns the clock of the processor in megahertz.
//
// There is nothing to read on this system.
func processorClock() int {
	return 0
}

// processorName returns the name of the processor.
//
// There is nothing to read on this system.
func processorName() string {
	return ""
}

// processorCount returns the number of the processors.
//
// There is nothing to read on this system.
func processorCount() int {
	return 0
}
