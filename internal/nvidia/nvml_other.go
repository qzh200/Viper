//go:build !windows

// Package nvidia reads the telemetry of the graphics card from the
// driver of the vendor.
//
// The management library of the driver belongs to Windows, so on every
// other system there is no telemetry to read. The names are the same on
// every system, so that the layers above do not have to know where they
// run.
package nvidia

import "errors"

// errNoDriver is the answer on a system without the driver.
var errNoDriver = errors.New("nvidia: 这个系统没有图形卡的驱动库")

// Telemetry is a snapshot of the graphics card.
//
// The fields are the same on every system, so that a caller does not have
// to build a different program for a different system.
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
	// PState is the performance state of the card. It is minus one when
	// the driver does not report it.
	PState int
	// Idle reports whether the card is doing nothing.
	Idle bool
}

// IsIdle reports whether the card sits idle.
//
// The driver does not always offer the performance state, so the load of
// the graphics processor is used to tell.
func (t Telemetry) IsIdle() bool {
	return t.UsagePercent == 0
}

// Available reports whether the telemetry can be read on this system.
//
// Without the driver of the vendor there is no telemetry, so the answer
// is no.
func Available() bool {
	return false
}

// Close releases the library of the driver.
//
// There is no library on this system, so there is nothing to release.
func Close() {}

// Read returns the telemetry of every card of the vendor.
//
// Without the driver of the vendor there is no card to read.
func Read() ([]Telemetry, error) {
	return nil, errNoDriver
}

// DriverVersion returns the version of the driver of the vendor.
//
// Without the driver of the vendor there is no version.
func DriverVersion() (string, error) {
	return "", errNoDriver
}
