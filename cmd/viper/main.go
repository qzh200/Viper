// viper 是 Acer 游戏笔记本（掠夺者 Predator / 暗影骑士 Nitro）的跨平台硬件控制中心命令行客户端。
package main

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"viper/internal/app"
	"viper/internal/hardware"
)

var (
	stdout io.Writer = os.Stdout
	stderr io.Writer = os.Stderr
)

var newApp = app.New

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(stderr, "viper:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		usage()
		return nil
	}

	for len(args) > 0 && strings.HasPrefix(args[0], "--") {
		switch args[0] {
		case "--service":
			if len(args) < 2 {
				return fmt.Errorf("--service needs an address, for example --service 127.0.0.1:46933")
			}
			app.SetServiceAddress(args[1])
			args = args[2:]
		case "--help":

			usage()
			return nil
		default:
			return fmt.Errorf("unknown option %q", args[0])
		}
	}
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
	case "gpu":
		return runGPU(args[1:])
	case "charge", "battery":
		return runCharge(args[1:])
	case "displaymode", "display":
		return runDisplayMode(args[1:])
	case "darkmode":
		return runDarkMode(args[1:])
	case "transparent":
		return runTransparent(args[1:])
	case "taskbar":
		return runTaskbar(args[1:])
	case "screensaver", "saver":
		return runScreenSaver(args[1:])
	case "bootsound", "sound":
		return runBootSound(args[1:])
	case "scenario", "scenes":
		return runScenario(args[1:])
	case "telemetry", "gpuinfo":
		return runTelemetry()
	case "powerstatus":
		return runPowerStatus()
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

func usage() {
	fmt.Fprintln(stdout, `viper - Acer gaming laptop control centre

Usage:
  viper status                      live temperatures and fan speeds
  viper powerstatus                  power supply and battery, no rights needed
  viper telemetry                    graphics card, no rights needed
  viper scenario list                show the saved scenes
  viper scenario save <name>         store the lighting and the fans
  viper scenario apply <name>        apply a scene
  viper scenario delete <name>       remove a scene
  viper scenario bind <profile> <name>  tie a scene to a profile
  viper scenario unbind <profile>    free a profile from its scene
  viper fans                        show the fan configuration
  viper fan auto                    hand the fans back to the controller
  viper fan max                     run the fans at full speed
  viper fan custom <cpu%> <gpu%>    run the fans at a fixed duty cycle
  viper fan coolboost <on|off>      toggle the boost fan table
  viper power                       show the performance profile
  viper power <mode>                set quiet|balanced|performance|turbo|eco
  viper lighting                    show the keyboard lighting
  viper lighting reset              restore the default lighting
  viper lighting set [options]      configure the keyboard lighting
  viper lighting timeout <on|off>   switch the backlight off on its own
  viper profile                     show overdrive and key lock state
  viper profile overdrive <on|off>  toggle the display overdrive
  viper profile winkey <on|off>     lock or unlock the Windows key
  viper profile stickykeys <on|off> switch the sticky keys on or off
  viper gpu                         show the graphics mode
  viper gpu <mode>                  set hybrid|discrete|auto
  viper charge                      show the charging settings
  viper charge limit <percent>      set the charging limit, 10, 20 or 30
  viper charge usb <on|off>         allow or forbid charging over USB
  viper displaymode                  show the way the machine drives the graphics
  viper displaymode <mode>           set hybrid|discrete|auto
  viper darkmode                     show the dark mode of the system
  viper darkmode <on|off>            switch the dark mode on or off
  viper transparent                  show the transparency of the windows
  viper transparent <on|off>         switch the transparency on or off
  viper taskbar                      show the automatic hiding of the taskbar
  viper taskbar <on|off>             hide the taskbar on its own
  viper screensaver                  show the screen saver
  viper screensaver <on|off>         switch the screen saver on or off
  viper screensaver timeout <n>      set the idle seconds before it starts
  viper bootsound                   show the sound at the start
  viper bootsound <on|off>          switch the sound at the start
  viper diag                        report the hardware capabilities

The display mode, the dark mode, the transparency, the taskbar and the
screen saver belong to the Quick Access service of the original software,
so the command needs that service to be running.

Lighting options:
  effect <name>      static|breathing|neon|wave|switch|scale|meteor|flash
  speed <n>          animation speed
  direction <n>      animation direction, 0 or 1
  brightness <0-100> overall brightness
  color <#RRGGBB>    one colour for every zone
  zone <n> <#RRGGBB> colour of a single zone, may be repeated
  all                finish the zone colours, every zone keeps its own colour

The options that are not mentioned keep the value the machine has now.`)
}

func runStatus() error {
	application, err := newApp()
	if err != nil {
		return err
	}
	defer application.Close()

	instance, err := application.Instance()
	if err != nil {
		return err
	}
	fmt.Fprintln(stdout, "instance:", instance)

	fmt.Fprintln(stdout, "route  :", application.Route())

	power, err := application.PowerStatus()
	if err != nil {
		return err
	}
	fmt.Fprintln(stdout)
	fmt.Fprintln(stdout, "  Power supply      :", power)
	if power.OnBattery {
		fmt.Fprintln(stdout, "  Battery           :", power.BatteryPercent, "%")
	}

	telemetry, err := application.Telemetry()
	if err != nil {
		return err
	}
	printTelemetry(telemetry)
	return nil
}

func printTelemetry(telemetry app.Telemetry) {
	monitor := telemetry.Monitor

	fmt.Fprintln(stdout)
	fmt.Fprintln(stdout, "  CPU temperature   :", monitor.CPUTemperatureC, "C")
	fmt.Fprintf(stdout, "  CPU usage         : %.1f %%\n", monitor.CPUUsagePercent)
	fmt.Fprintf(stdout, "  CPU clock         : %d MHz (max %d MHz)\n", monitor.CPUFrequencyMHz, monitor.CPUMaxFrequencyMHz)
	fmt.Fprintln(stdout, "  CPU fan speed     :", monitor.CPUFanRPM, "RPM")
	fmt.Fprintln(stdout, "  GPU temperature   :", monitor.GPUTemperatureC, "C")
	fmt.Fprintln(stdout, "  GPU usage         :", monitor.GPUUsagePercent, "%")
	fmt.Fprintf(stdout, "  GPU clock         : %d MHz (max %d MHz)\n", monitor.GPUFrequencyMHz, monitor.GPUMaxFrequencyMHz)
	fmt.Fprintln(stdout, "  GPU fan speed     :", monitor.GPUFanRPM, "RPM")
	fmt.Fprintln(stdout, "  Motherboard / VRM :", monitor.SystemTempC, "C")
	fmt.Fprintf(stdout, "  Memory            : %.1f %% of %.0f MB @ %d MHz\n",
		monitor.RAMUsagePercent, monitor.RAMTotalMB, monitor.RAMFrequencyMHz)
	fmt.Fprintf(stdout, "  Network           : wifi down %.1f up %.1f kbps, ethernet down %.1f up %.1f kbps\n",
		monitor.WIFIDownloadKbps, monitor.WIFIUploadKbps,
		monitor.EthernetDownloadKbps, monitor.EthernetUploadKbps)

	if len(monitor.CPUCoreFrequencyMHz) > 0 {
		clocks := monitor.CPUCoreFrequencyMHz
		fmt.Fprintf(stdout, "  Core clocks       : %d cores, %d to %d MHz\n",
			len(clocks), minOf(clocks), maxOf(clocks))
	}

	for _, card := range telemetry.Cards {
		fmt.Fprintln(stdout)
		fmt.Fprintln(stdout, "  Graphics card     :", card.Name)
		fmt.Fprintln(stdout, "  Memory clock      :", card.MemoryClockMHz, "MHz")
		fmt.Fprintln(stdout, "  Video memory      :", formatBytes(card.MemoryUsedBytes), "of", formatBytes(card.MemoryTotalBytes))
		if card.IsIdle() {
			fmt.Fprintln(stdout, "  The card is idle")
		}
	}
}

func minOf(values []int) int {
	smallest := values[0]
	for _, value := range values {
		if value < smallest {
			smallest = value
		}
	}
	return smallest
}

func maxOf(values []int) int {
	largest := values[0]
	for _, value := range values {
		if value > largest {
			largest = value
		}
	}
	return largest
}

func formatBytes(value uint64) string {
	switch {
	case value >= 1<<30:
		return fmt.Sprintf("%.1f GB", float64(value)/(1<<30))
	case value >= 1<<20:
		return fmt.Sprintf("%.0f MB", float64(value)/(1<<20))
	case value >= 1<<10:
		return fmt.Sprintf("%.0f kB", float64(value)/(1<<10))
	default:
		return fmt.Sprintf("%d bytes", value)
	}
}

func runScenario(args []string) error {
	application, err := newApp()
	if err != nil {
		return err
	}
	defer application.Close()

	if len(args) == 0 {
		return listScenarios(application)
	}

	switch args[0] {
	case "list":
		return listScenarios(application)
	case "save":
		if len(args) < 2 {
			return fmt.Errorf("usage: viper scenario save <name>")
		}
		if err := application.SaveScenario(args[1]); err != nil {
			return err
		}
		fmt.Fprintln(stdout, "scene saved:", args[1])
		return nil
	case "apply":
		if len(args) < 2 {
			return fmt.Errorf("usage: viper scenario apply <name>")
		}
		if err := application.ApplyScenario(args[1]); err != nil {
			return err
		}
		fmt.Fprintln(stdout, "scene applied:", args[1])
		return nil
	case "delete", "remove":
		if len(args) < 2 {
			return fmt.Errorf("usage: viper scenario delete <name>")
		}
		if err := application.DeleteScenario(args[1]); err != nil {
			return err
		}
		fmt.Fprintln(stdout, "scene deleted:", args[1])
		return nil
	case "bind":
		if len(args) < 3 {
			return fmt.Errorf("usage: viper scenario bind <profile> <name>")
		}
		mode, ok := hardware.ParsePowerMode(strings.ToLower(args[1]))
		if !ok {
			return fmt.Errorf("unknown performance profile %q", args[1])
		}
		if err := application.BindScenario(mode.String(), args[2]); err != nil {
			return err
		}
		fmt.Fprintf(stdout, "scene %q is bound to the profile %s\n", args[2], mode)
		return nil
	case "unbind":
		if len(args) < 2 {
			return fmt.Errorf("usage: viper scenario unbind <profile>")
		}
		mode, ok := hardware.ParsePowerMode(strings.ToLower(args[1]))
		if !ok {
			return fmt.Errorf("unknown performance profile %q", args[1])
		}
		if err := application.UnbindScenario(mode.String()); err != nil {
			return err
		}
		fmt.Fprintln(stdout, "the profile", mode, "has no scene")
		return nil
	default:
		return fmt.Errorf("unknown scenario command %q", args[0])
	}
}

func listScenarios(application *app.App) error {
	scenarios := application.Scenarios()
	if len(scenarios) == 0 {
		fmt.Fprintln(stdout, "  no scene is saved")
	} else {
		for name, scenario := range scenarios {
			fmt.Fprintf(stdout, "  %-16s lighting %s at %d%%, fans %s\n", name,
				hardware.LightingEffect(scenario.Lighting.Effect),
				scenario.Lighting.Brightness,
				hardware.FanMode(scenario.Fans.CPUMode))
		}
	}

	bindings := application.Bindings()
	if len(bindings) == 0 {
		return nil
	}
	fmt.Fprintln(stdout)
	fmt.Fprintln(stdout, "  bound scenes:")
	for profile, name := range bindings {
		fmt.Fprintf(stdout, "    %-12s -> %s\n", profile, name)
	}
	return nil
}

func runTelemetry() error {
	application, err := newApp()
	if err != nil {
		return err
	}
	defer application.Close()

	telemetry, err := application.Telemetry()
	if err != nil {
		return err
	}
	printTelemetry(telemetry)
	return nil
}

func runPowerStatus() error {
	application, err := newApp()
	if err != nil {
		return err
	}
	defer application.Close()

	power, err := application.PowerStatus()
	if err != nil {
		return err
	}
	fmt.Fprintln(stdout, "  Power supply      :", power)
	if power.OnBattery {
		fmt.Fprintln(stdout, "  Battery           :", power.BatteryPercent, "%")
	}
	if power.BatteryLow {
		fmt.Fprintln(stdout, "  The battery is running low")
	}
	if power.BatteryCritical {
		fmt.Fprintln(stdout, "  The battery is almost empty")
	}
	return nil
}

func runFan(args []string) error {
	application, err := newApp()
	if err != nil {
		return err
	}
	defer application.Close()

	if len(args) == 0 {
		state, err := application.Fans()
		if err != nil {
			return err
		}
		fmt.Fprintln(stdout, "  CPU fan    :", describeFan(state.CPUMode, state.CPUPercent))
		fmt.Fprintln(stdout, "  GPU fan    :", describeGPUFan(state))
		fmt.Fprintln(stdout, "  CoolBoost :", onOff(state.CoolBoost))
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
		if err := application.SetCoolBoost(enabled); err != nil {
			return err
		}
		fmt.Fprintln(stdout, "CoolBoost:", onOff(enabled))
		return nil
	case "auto", "max":
		mode := hardware.FanAuto
		if args[0] == "max" {
			mode = hardware.FanMax
		}
		return applyFan(application, hardware.FanConfig{CPUMode: mode, GPUMode: mode})
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
		return applyFan(application, hardware.FanConfig{
			CPUMode:    hardware.FanCustom,
			GPUMode:    hardware.FanCustom,
			CPUPercent: cpu,
			GPUPercent: gpu,
		})
	default:
		return fmt.Errorf("unknown fan command %q", args[0])
	}
}

func applyFan(application *app.App, config hardware.FanConfig) error {
	if err := application.SetFans(config); err != nil {
		return err
	}
	state, err := application.Fans()
	if err != nil {
		return err
	}
	fmt.Fprintln(stdout, "  CPU fan    :", describeFan(state.CPUMode, state.CPUPercent))
	fmt.Fprintln(stdout, "  GPU fan    :", describeGPUFan(state))
	return nil
}

func describeFan(mode hardware.FanMode, percent int) string {
	if mode == hardware.FanCustom {
		return fmt.Sprintf("%s (%d%%)", mode, percent)
	}
	return mode.String()
}

func describeGPUFan(state app.FanStatus) string {
	text := describeFan(state.GPUMode, state.GPUPercent)
	if !state.GPUModeReadable {
		text += " (not readable from the machine, this is the last value Viper wrote)"
	}
	return text
}

func runPower(args []string) error {
	application, err := newApp()
	if err != nil {
		return err
	}
	defer application.Close()

	if len(args) == 0 {
		mode, err := application.PowerMode()
		if err != nil {
			return err
		}
		fmt.Fprintln(stdout, "  performance profile:", mode)
		return nil
	}

	mode, ok := hardware.ParsePowerMode(strings.ToLower(args[0]))
	if !ok {
		return fmt.Errorf("unknown performance profile %q", args[0])
	}
	if mode == hardware.PowerTurbo && !hardware.IsPredator() {
		return fmt.Errorf("turbo is exclusive to the Predator series")
	}
	if err := application.SetPowerMode(mode); err != nil {
		return err
	}
	current, err := application.PowerMode()
	if err != nil {
		return err
	}
	fmt.Fprintln(stdout, "  performance profile:", current)
	return nil
}

func runLighting(args []string) error {
	application, err := newApp()
	if err != nil {
		return err
	}
	defer application.Close()

	if len(args) > 0 && args[0] == "reset" {
		if err := application.ResetLighting(); err != nil {
			return err
		}
		fmt.Fprintln(stdout, "lighting reset to the default")
		return nil
	}

	if len(args) > 0 && args[0] == "timeout" {
		if len(args) < 2 {
			return fmt.Errorf("usage: viper lighting timeout <on|off>")
		}
		enabled, err := parseOnOff(args[1])
		if err != nil {
			return err
		}
		if err := application.SetBacklightTimeout(enabled); err != nil {
			return err
		}
		fmt.Fprintln(stdout, "backlight timeout:", onOff(enabled))
		return nil
	}

	if len(args) == 0 || args[0] != "set" {
		return showLighting(application)
	}

	base, err := application.Lighting()
	if err != nil {
		base = application.StoredLighting()
	}

	options, err := parseLightingOptions(base, args[1:])
	if err != nil {
		return err
	}
	if err := application.SetLighting(*options); err != nil {
		return err
	}
	fmt.Fprintln(stdout, "lighting applied")
	return nil
}

func showLighting(application *app.App) error {
	config, err := application.Lighting()
	if err != nil {
		fmt.Fprintln(stderr, "viper:", err)
		fmt.Fprintln(stdout, "  machine     : not readable, showing what Viper wrote last")
		config = application.StoredLighting()
	} else {
		fmt.Fprintf(stdout, "  brightness  : %d\n", config.Brightness)
	}
	fmt.Fprintf(stdout, "  effect      : %s\n", config.Effect)
	fmt.Fprintf(stdout, "  speed       : %d\n", config.Speed)
	fmt.Fprintf(stdout, "  direction   : %d\n", config.Direction)
	for i, zone := range config.Zones {
		fmt.Fprintf(stdout, "  zone %d      : #%02X%02X%02X\n", i+1, zone.R, zone.G, zone.B)
	}
	timeout, err := application.BacklightTimeout()
	if err == nil {
		fmt.Fprintln(stdout, "  timeout     :", onOff(timeout))
	}
	fmt.Fprintln(stdout, "  settings    :", application.SettingsPath())
	return nil
}

func parseLightingOptions(base hardware.RGBConfig, args []string) (*hardware.RGBConfig, error) {
	config := base
	zoneColours := map[int]hardware.RGBColor{}

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "effect":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("effect needs a value")
			}
			effect, ok := hardware.ParseLightingEffect(strings.ToLower(args[i+1]))
			if !ok {
				return nil, fmt.Errorf("unknown lighting effect %q", args[i+1])
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
		case "color", "colour":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("color needs a value")
			}
			colour, err := parseColor(args[i+1])
			if err != nil {
				return nil, err
			}
			for zone := range config.Zones {
				config.Zones[zone] = colour
			}
			config.PerZone = false
			i++
		case "zone":
			if i+2 >= len(args) {
				return nil, fmt.Errorf("zone needs a number and a colour")
			}
			zone, err := strconv.Atoi(args[i+1])
			if err != nil {
				return nil, err
			}
			if zone < 1 || zone > hardware.LightingZones {
				return nil, fmt.Errorf("zone %d is out of range", zone)
			}
			colour, err := parseColor(args[i+2])
			if err != nil {
				return nil, err
			}
			zoneColours[zone] = colour
			config.PerZone = true
			i += 2
		case "all":

			for zone := range config.Zones {
				if colour, ok := zoneColours[zone+1]; ok {
					config.Zones[zone] = colour
				}
			}
			config.PerZone = len(zoneColours) > 0
		default:
			return nil, fmt.Errorf("unknown lighting option %q", args[i])
		}
	}

	for zone, colour := range zoneColours {
		config.Zones[zone-1] = colour
	}
	return &config, nil
}

func parseColor(text string) (hardware.RGBColor, error) {
	hex := strings.TrimPrefix(strings.TrimSpace(text), "#")
	if len(hex) != 6 {
		return hardware.RGBColor{}, fmt.Errorf("colour %q must be six hex digits", text)
	}
	value, err := strconv.ParseUint(hex, 16, 32)
	if err != nil {
		return hardware.RGBColor{}, fmt.Errorf("colour %q: %w", text, err)
	}
	return hardware.RGBColor{
		R: uint8(value >> 16),
		G: uint8(value >> 8),
		B: uint8(value),
	}, nil
}

func runProfile(args []string) error {
	application, err := newApp()
	if err != nil {
		return err
	}
	defer application.Close()

	if len(args) == 0 {
		overdrive, err := application.LCDOverdrive()
		if err != nil {
			return err
		}
		fmt.Fprintln(stdout, "  LCD overdrive :", onOff(overdrive))

		locked, err := application.WinKeyLocked()
		if err != nil {
			fmt.Fprintln(stderr, "viper: 读不到 Windows 键的状态（需要管理员权限）:", err)
		} else {
			fmt.Fprintln(stdout, "  Windows key   :", onOff(locked))
		}

		sticky, err := application.StickyKeys()
		if err != nil {
			return err
		}
		fmt.Fprintln(stdout, "  Sticky keys   :", onOff(sticky))
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
		if err := application.SetLCDOverdrive(enabled); err != nil {
			return err
		}
		fmt.Fprintln(stdout, "LCD overdrive:", onOff(enabled))
		return nil
	case "winkey", "win":
		if len(args) < 2 {
			return fmt.Errorf("usage: viper profile winkey <on|off>")
		}
		locked, err := parseOnOff(args[1])
		if err != nil {
			return err
		}
		if err := application.SetWinKeyLocked(locked); err != nil {
			return err
		}
		fmt.Fprintln(stdout, "Windows key:", onOff(locked))
		return nil
	case "stickykeys", "sticky":
		if len(args) < 2 {
			return fmt.Errorf("usage: viper profile stickykeys <on|off>")
		}
		enabled, err := parseOnOff(args[1])
		if err != nil {
			return err
		}
		if err := application.SetStickyKeys(enabled); err != nil {
			return err
		}
		fmt.Fprintln(stdout, "sticky keys:", onOff(enabled))
		return nil
	default:
		return fmt.Errorf("unknown profile command %q", args[0])
	}
}

func runGPU(args []string) error {
	application, err := newApp()
	if err != nil {
		return err
	}
	defer application.Close()

	if len(args) == 0 {
		mode, err := application.GPUMode()
		if err != nil {
			return err
		}
		fmt.Fprintln(stdout, "  graphics mode:", mode)
		return nil
	}

	mode, ok := hardware.ParseGPUMode(strings.ToLower(args[0]))
	if !ok {
		return fmt.Errorf("unknown graphics mode %q", args[0])
	}
	if err := application.SetGPUMode(mode); err != nil {
		return err
	}
	fmt.Fprintln(stdout, "  graphics mode:", mode, "(takes effect after a restart)")
	return nil
}

func runCharge(args []string) error {
	application, err := newApp()
	if err != nil {
		return err
	}
	defer application.Close()

	if len(args) == 0 {
		limit, err := application.USBChargeLimit()
		if err != nil {
			return err
		}
		enabled, err := application.USBChargeEnabled()
		if err != nil {
			return err
		}
		fmt.Fprintf(stdout, "  charging limit  : %d%%\n", limit)
		fmt.Fprintln(stdout, "  charging over USB:", onOff(enabled))
		return nil
	}

	switch args[0] {
	case "limit":
		if len(args) < 2 {
			return fmt.Errorf("usage: viper charge limit <10|20|30>")
		}
		percent, err := strconv.Atoi(args[1])
		if err != nil {
			return fmt.Errorf("the charging limit needs a percentage: %w", err)
		}
		if err := application.SetUSBChargeLimit(percent); err != nil {
			return err
		}
		fmt.Fprintf(stdout, "  charging limit  : %d%%\n", percent)
		return nil
	case "usb":
		if len(args) < 2 {
			return fmt.Errorf("usage: viper charge usb <on|off>")
		}
		enabled, err := parseOnOff(args[1])
		if err != nil {
			return err
		}
		if err := application.SetUSBChargeEnabled(enabled); err != nil {
			return err
		}
		fmt.Fprintln(stdout, "  charging over USB:", onOff(enabled))
		return nil
	default:
		return fmt.Errorf("unknown charge command %q", args[0])
	}
}

func runBootSound(args []string) error {
	application, err := newApp()
	if err != nil {
		return err
	}
	defer application.Close()

	if len(args) == 0 {
		enabled, err := application.BootSound()
		if err != nil {
			return err
		}
		fmt.Fprintln(stdout, "  boot sound:", onOff(enabled))
		return nil
	}

	enabled, err := parseOnOff(args[0])
	if err != nil {
		return err
	}
	if err := application.SetBootSound(enabled); err != nil {
		return err
	}
	fmt.Fprintln(stdout, "  boot sound:", onOff(enabled))
	return nil
}

func runDiag() error {
	application, err := newApp()
	if err != nil {
		return err
	}
	defer application.Close()

	instance, err := application.Instance()
	if err != nil {
		return err
	}
	fmt.Fprintln(stdout, "  instance        :", instance)
	fmt.Fprintln(stdout, "  route           :", application.Route())

	identity, err := application.Identity()
	if err != nil {
		return err
	}
	fmt.Fprintln(stdout, "  model           :", identity.Product)
	fmt.Fprintln(stdout, "  family          :", identity.Family)
	isPredator := hardware.IsPredatorModel(identity.Product, identity.Family)
	fmt.Fprintln(stdout, "  Predator series :", onOff(isPredator))

	modes := make([]string, 0, 4)
	for _, mode := range application.AvailablePowerModes() {
		modes = append(modes, mode.String())
	}
	fmt.Fprintln(stdout, "  profiles        :", strings.Join(modes, ", "))

	mode, err := application.PowerMode()
	report("power profile", mode, err)

	state, fanErr := application.Fans()
	if fanErr != nil {
		report("fans", nil, fanErr)
	} else {
		fmt.Fprintln(stdout, "  CPU fan         :", describeFan(state.CPUMode, state.CPUPercent))
		fmt.Fprintln(stdout, "  GPU fan         :", describeFan(state.GPUMode, state.GPUPercent))
		fmt.Fprintln(stdout, "  CoolBoost       :", onOff(state.CoolBoost))
	}

	discrete, err := application.DiscreteGPU()
	report("discrete GPU", onOff(discrete), err)

	gpuOC, cpuOC, err := application.OverclockCapability()
	if err != nil {
		report("overclock", nil, err)
	} else {
		fmt.Fprintln(stdout, "  GPU overclock   :", onOff(gpuOC))
		fmt.Fprintln(stdout, "  CPU overclock   :", onOff(cpuOC))
	}

	lighting, err := application.Lighting()
	if err != nil {
		report("key lighting", nil, err)
	} else {
		fmt.Fprintln(stdout, "  key brightness  :", lighting.Brightness)
		fmt.Fprintln(stdout, "  key effect      :", lighting.Effect)
	}

	gpuMode, err := application.GPUMode()
	report("graphics mode", gpuMode, err)

	bootSound, err := application.BootSound()
	report("boot sound", onOff(bootSound), err)

	sticky, err := application.StickyKeys()
	report("sticky keys", onOff(sticky), err)

	locked, err := application.WinKeyLocked()
	report("Windows key", onOff(locked), err)

	fmt.Fprintln(stdout, "  settings        :", application.SettingsPath())
	return nil
}

func report(name string, value any, err error) {
	if err != nil {
		fmt.Fprintf(stdout, "  %-16s: 读不到（%v）\n", name, err)
		return
	}
	fmt.Fprintf(stdout, "  %-16s: %v\n", name, value)
}

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

func onOff(value bool) string {
	if value {
		return "on"
	}
	return "off"
}
