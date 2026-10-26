// Command viper is the Viper control centre for Acer gaming laptops.
//
// Every subcommand talks to the hardware directly through the ACPI WMI
// provider, so the process has to run with administrative rights.
package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

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
	case "fan", "fans":
		return runFan(args[1:])
	case "power", "mode":
		return runPower(args[1:])
	case "lighting", "light", "keyboard":
		return runLighting(args[1:])
	case "profile", "lcd":
		return runProfile(args[1:])
	case "diag", "diagnostics":
		return runDiag()
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
  viper status                     live temperatures and fan speeds
  viper fans                       show the fan configuration
  viper fan auto                   hand the fans back to the controller
  viper fan max                    run the fans at full speed
  viper fan custom <cpu%> <gpu%>   run the fans at a fixed duty cycle
  viper fan coolboost <on|off>     toggle the boost fan table
  viper power                       show the performance profile
  viper power <mode>                set quiet|balanced|performance|turbo
  viper lighting                    show the keyboard lighting
  viper lighting set <options>      configure the keyboard lighting
  viper profile                     show overdrive and key lock state
  viper profile overdrive <on|off>  toggle the display overdrive
  viper profile winkey <on|off>     lock or unlock the Windows key
  viper diag                        report the hardware capabilities`)
}

// backend opens the ACPI provider.
func backend() (*hardware.Backend, error) {
	return hardware.NewBackend()
}

// runStatus prints a live telemetry snapshot.
func runStatus() error {
	b, err := backend()
	if err != nil {
		return err
	}
	defer b.Close()

	fmt.Println("instance:", b.Instance())

	sensors, err := b.Sensors()
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

// runFan shows or changes the fan configuration.
func runFan(args []string) error {
	b, err := backend()
	if err != nil {
		return err
	}
	defer b.Close()

	if len(args) == 0 {
		state, err := b.Fans()
		if err != nil {
			return err
		}
		fmt.Println("  CPU fan    :", describeFan(state.CPUMode, state.CPUPercent))
		fmt.Println("  GPU fan    :", describeFan(state.GPUMode, state.GPUPercent))
		fmt.Println("  CoolBoost :", onOff(state.CoolBoost))
		return nil
	}

	switch args[0] {
	case "coolboost", "cb":
		if len(args) < 2 {
			return fmt.Errorf("usage: viper fan coolboost <on|off>")
		}
		enabled, err := parseOnOff(args[1])
		if err != nil {
			return err
		}
		if err := b.CoolBoost(enabled); err != nil {
			return err
		}
		fmt.Println("CoolBoost:", onOff(enabled))
		return nil
	case "auto", "max":
		mode := hardware.FanAuto
		if args[0] == "max" {
			mode = hardware.FanMax
		}
		return applyFan(b, hardware.FanConfig{CPUMode: mode, GPUMode: mode})
	case "custom":
		if len(args) < 3 {
			return fmt.Errorf("usage: viper fan custom <cpu%%> <gpu%%>")
		}
		cpu, err := strconv.Atoi(args[1])
		if err != nil {
			return err
		}
		gpu, err := strconv.Atoi(args[2])
		if err != nil {
			return err
		}
		return applyFan(b, hardware.FanConfig{
			CPUMode:    hardware.FanCustom,
			GPUMode:    hardware.FanCustom,
			CPUPercent: cpu,
			GPUPercent: gpu,
		})
	default:
		return fmt.Errorf("unknown fan command %q", args[0])
	}
}

// applyFan applies a fan configuration and reports the result.
func applyFan(b *hardware.Backend, config hardware.FanConfig) error {
	if err := b.SetFans(config); err != nil {
		return err
	}
	state, err := b.Fans()
	if err != nil {
		return err
	}
	fmt.Println("  CPU fan    :", describeFan(state.CPUMode, state.CPUPercent))
	fmt.Println("  GPU fan    :", describeFan(state.GPUMode, state.GPUPercent))
	return nil
}

// describeFan renders a fan behaviour. The duty cycle is only meaningful
// in custom mode, in automatic mode the controller runs its own curve.
func describeFan(mode hardware.FanMode, percent int) string {
	if mode == hardware.FanCustom {
		return fmt.Sprintf("%s (%d%%)", mode, percent)
	}
	return fmt.Sprintf("%s (controller curve)", mode)
}

// runPower shows or changes the performance profile.
func runPower(args []string) error {
	b, err := backend()
	if err != nil {
		return err
	}
	defer b.Close()

	if len(args) == 0 {
		mode, err := b.PowerMode()
		if err != nil {
			return err
		}
		fmt.Println("  performance profile:", mode)
		return nil
	}

	mode, ok := hardware.ParsePowerMode(strings.ToLower(args[0]))
	if !ok {
		return fmt.Errorf("unknown performance profile %q", args[0])
	}
	if mode == hardware.PowerTurbo && !hardware.IsPredator() {
		return fmt.Errorf("turbo is exclusive to the Predator series")
	}
	if err := b.SetPowerMode(mode); err != nil {
		return err
	}
	current, err := b.PowerMode()
	if err != nil {
		return err
	}
	fmt.Println("  performance profile:", current)
	return nil
}

// runLighting shows or changes the keyboard lighting.
func runLighting(args []string) error {
	b, err := backend()
	if err != nil {
		return err
	}
	defer b.Close()

	if len(args) == 0 || args[0] != "set" {
		config, err := b.Lighting()
		if err != nil {
			return err
		}
		fmt.Printf("  effect      : %d\n", config.Effect)
		fmt.Printf("  speed       : %d\n", config.Speed)
		fmt.Printf("  direction   : %d\n", config.Direction)
		fmt.Printf("  brightness  : %d\n", config.Brightness)
		for i, zone := range config.Zones {
			fmt.Printf("  zone %d      : #%02X%02X%02X\n", i+1, zone.R, zone.G, zone.B)
		}
		return nil
	}

	options, err := parseLightingOptions(args[1:])
	if err != nil {
		return err
	}
	if err := b.SetLighting(*options); err != nil {
		return err
	}
	fmt.Println("lighting applied")
	return nil
}

// parseLightingOptions parses the arguments of "viper lighting set".
func parseLightingOptions(args []string) (*hardware.RGBConfig, error) {
	config := &hardware.RGBConfig{Effect: hardware.EffectStatic, Speed: 3, Brightness: 50}

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "effect":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("effect needs a value")
			}
			effect, err := parseEffect(args[i+1])
			if err != nil {
				return nil, err
			}
			config.Effect = effect
			i++
		case "speed":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("speed needs a value")
			}
			speed, err := strconv.Atoi(args[i+1])
			if err != nil {
				return nil, err
			}
			config.Speed = uint64(speed)
			i++
		case "direction":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("direction needs a value")
			}
			direction, err := strconv.Atoi(args[i+1])
			if err != nil {
				return nil, err
			}
			config.Direction = uint64(direction)
			i++
		case "brightness":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("brightness needs a value")
			}
			brightness, err := strconv.Atoi(args[i+1])
			if err != nil {
				return nil, err
			}
			config.Brightness = uint64(brightness)
			i++
		case "color":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("color needs a value")
			}
			zones, err := parseColors(args[i+1])
			if err != nil {
				return nil, err
			}
			config.Zones = zones
			i++
		default:
			return nil, fmt.Errorf("unknown lighting option %q", args[i])
		}
	}
	return config, nil
}

// parseEffect maps a name onto a lighting effect.
func parseEffect(name string) (uint64, error) {
	switch strings.ToLower(name) {
	case "static":
		return hardware.EffectStatic, nil
	case "breathing", "breathe":
		return hardware.EffectBreathing, nil
	case "wave":
		return hardware.EffectWave, nil
	case "lightning", "flash":
		return hardware.EffectLightning, nil
	case "spectrum", "rainbow":
		return hardware.EffectSpectrum, nil
	default:
		return 0, fmt.Errorf("unknown lighting effect %q", name)
	}
}

// parseColors parses a comma separated list of hex colours.
func parseColors(list string) ([4]hardware.RGBColor, error) {
	var zones [4]hardware.RGBColor
	parts := strings.Split(list, ",")
	if len(parts) > 4 {
		return zones, fmt.Errorf("at most four zone colours are accepted")
	}
	for i, part := range parts {
		hex := strings.TrimPrefix(strings.TrimSpace(part), "#")
		if len(hex) != 6 {
			return zones, fmt.Errorf("colour %q must be six hex digits", part)
		}
		value, err := strconv.ParseUint(hex, 16, 32)
		if err != nil {
			return zones, fmt.Errorf("colour %q: %w", part, err)
		}
		zones[i] = hardware.RGBColor{
			R: uint8(value >> 16),
			G: uint8(value >> 8),
			B: uint8(value),
		}
	}
	return zones, nil
}

// runProfile shows or changes the display overdrive and the key lock.
func runProfile(args []string) error {
	b, err := backend()
	if err != nil {
		return err
	}
	defer b.Close()

	if len(args) == 0 {
		overdrive, err := b.LCDOverdrive()
		if err != nil {
			return err
		}
		locked, err := b.WinKeyLocked()
		if err != nil {
			return err
		}
		fmt.Println("  LCD overdrive :", onOff(overdrive))
		fmt.Println("  Windows key   :", onOff(locked))
		return nil
	}

	switch args[0] {
	case "overdrive", "od":
		if len(args) < 2 {
			return fmt.Errorf("usage: viper profile overdrive <on|off>")
		}
		enabled, err := parseOnOff(args[1])
		if err != nil {
			return err
		}
		if err := b.SetLCDOverdrive(enabled); err != nil {
			return err
		}
		fmt.Println("LCD overdrive:", onOff(enabled))
		return nil
	case "winkey", "win":
		if len(args) < 2 {
			return fmt.Errorf("usage: viper profile winkey <on|off>")
		}
		locked, err := parseOnOff(args[1])
		if err != nil {
			return err
		}
		if err := b.SetWinKeyLocked(locked); err != nil {
			return err
		}
		fmt.Println("Windows key:", onOff(locked))
		return nil
	default:
		return fmt.Errorf("unknown profile command %q", args[0])
	}
}

// runDiag reports the capabilities of the machine.
func runDiag() error {
	b, err := backend()
	if err != nil {
		return err
	}
	defer b.Close()

	fmt.Println("  instance        :", b.Instance())

	identity, err := hardware.SystemInfo()
	if err != nil {
		return err
	}
	fmt.Println("  model           :", identity.Product)
	fmt.Println("  family          :", identity.Family)
	isPredator := hardware.IsPredatorModel(identity.Product, identity.Family)
	fmt.Println("  Predator series :", onOff(isPredator))

	modes := make([]string, 0, 4)
	for _, mode := range hardware.AvailablePowerModes(isPredator) {
		modes = append(modes, mode.String())
	}
	fmt.Println("  profiles        :", strings.Join(modes, ", "))

	mode, err := b.PowerMode()
	if err != nil {
		return err
	}
	fmt.Println("  power profile   :", mode)

	discrete, err := b.DiscreteGPU()
	if err != nil {
		return err
	}
	fmt.Println("  discrete GPU    :", onOff(discrete))

	gpuOC, cpuOC, err := b.OverclockCapability()
	if err != nil {
		return err
	}
	fmt.Println("  GPU overclock   :", onOff(gpuOC))
	fmt.Println("  CPU overclock   :", onOff(cpuOC))

	coolBoost, err := b.CoolBoostEnabled()
	if err != nil {
		return err
	}
	fmt.Println("  CoolBoost       :", onOff(coolBoost))

	backlight, err := b.Backlight()
	if err != nil {
		return err
	}
	fmt.Println("  backlight       :", backlight.Brightness, "timeout:", onOff(backlight.Timeout))
	return nil
}

// parseOnOff parses an on or off argument.
func parseOnOff(value string) (bool, error) {
	switch strings.ToLower(value) {
	case "on", "true", "1", "yes", "enable":
		return true, nil
	case "off", "false", "0", "no", "disable":
		return false, nil
	default:
		return false, fmt.Errorf("expected on or off, got %q", value)
	}
}

// onOff renders a boolean for display.
func onOff(value bool) string {
	if value {
		return "on"
	}
	return "off"
}
