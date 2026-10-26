// Command viper is the Viper control centre for Acer gaming laptops.
//
// Every subcommand talks to the hardware directly through the ACPI WMI
// provider, so the process has to run with administrative rights.
package main

import (
	"fmt"
	"os"

	"viper/internal/hardware"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "viper:", err)
		os.Exit(1)
	}
}

// run dispatches a subcommand.
func run(args []string) error {
	if len(args) == 0 {
		usage()
		return nil
	}

	switch args[0] {
	case "status":
		return runStatus()
	case "help", "-h", "--help":
		usage()
		return nil
	default:
		usage()
		return fmt.Errorf("unknown command %q", args[0])
	}
}

// usage prints the command overview.
func usage() {
	fmt.Println(`viper - Acer gaming laptop control centre

Usage:
  viper status    show live temperatures and fan speeds`)
}

// runStatus prints a live telemetry snapshot.
func runStatus() error {
	backend, err := hardware.NewBackend()
	if err != nil {
		return err
	}
	defer backend.Close()

	fmt.Println("instance:", backend.Instance())

	sensors, err := backend.Sensors()
	if err != nil {
		return err
	}

	fmt.Println()
	fmt.Println("  CPU temperature   :", sensors.CPUTempC, "C")
	fmt.Println("  CPU fan speed     :", sensors.CPUFanRPM, "RPM")
	fmt.Println("  GPU temperature   :", sensors.GPUTempC, "C")
	fmt.Println("  GPU fan speed     :", sensors.GPUFanRPM, "RPM")
	fmt.Println("  Motherboard / VRM :", sensors.SystemTempC, "C")
	return nil
}
