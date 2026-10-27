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

	"viper/internal/app"
	"viper/internal/hardware"
	"viper/internal/nvidia"
	"viper/internal/quickaccess"
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
	case "gpu":
		return runGPU(args[1:])
	case "bluelight", "blueshield":
		return runBluelight(args[1:])
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

// usage prints the command overview.
func usage() {
	fmt.Println(`viper - Acer gaming laptop control centre

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
  viper bluelight                   show the blue light shield
  viper bluelight <level>           set the shield, 0 to 4
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
  all                apply the colours to every zone

Blue light shield levels:
  0 disabled   1 low   2 medium   3 high   4 long hour`)
}

// runStatus prints a live telemetry snapshot.
func runStatus() error {
	application, err := app.New()
	if err != nil {
		return err
	}
	defer application.Close()

	instance, err := application.Instance()
	if err != nil {
		return err
	}
	fmt.Println("instance:", instance)
	// 走的哪条路要报出来：原厂服务那一路不用管理员权限，固件那一路要。
	fmt.Println("route  :", application.Route())

	// The power supply comes from the kernel of Windows, it is shown
	// first because it is there even when the hardware needs the rights
	// of an administrator.
	power, err := application.PowerStatus()
	if err != nil {
		return err
	}
	fmt.Println()
	fmt.Println("  Power supply      :", power)
	if power.OnBattery {
		fmt.Println("  Battery           :", power.BatteryPercent, "%")
	}

	sensors, err := application.Sensors()
	if err != nil {
		return err
	}

	fmt.Println()
	fmt.Println("  CPU temperature   :", sensors.CPUTempC, "C")
	fmt.Println("  CPU fan speed     :", sensors.CPUFanRPM, "RPM")
	fmt.Println("  GPU temperature   :", sensors.GPUTempC, "C")
	fmt.Println("  GPU fan speed     :", sensors.GPUFanRPM, "RPM")
	fmt.Println("  Motherboard / VRM :", sensors.SystemTempC, "C")

	cards, err := application.GPUTelemetry()
	if err != nil {
		return err
	}
	for _, card := range cards {
		printCard(card)
	}
	return nil
}

// printCard shows the state of a graphics card.
func printCard(card nvidia.Telemetry) {
	fmt.Println()
	fmt.Println("  Graphics card     :", card.Name)
	fmt.Println("  Graphics usage    :", card.UsagePercent, "%")
	fmt.Println("  Graphics clock    :", card.GraphicsClockMHz, "MHz (max", card.MaxGraphicsClockMHz, "MHz)")
	fmt.Println("  Memory clock      :", card.MemoryClockMHz, "MHz")
	fmt.Println("  Video memory      :", formatBytes(card.MemoryUsedBytes), "of", formatBytes(card.MemoryTotalBytes))
	if card.IsIdle() {
		fmt.Println("  The card is idle")
	}
}

// formatBytes renders a number of bytes in a readable way.
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

// runScenario shows or changes the scenes.
//
// A scene holds the lighting and the fans, it can be bound to a
// performance profile so that the profile brings its own look.
func runScenario(args []string) error {
	application, err := app.New()
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
		fmt.Println("scene saved:", args[1])
		return nil
	case "apply":
		if len(args) < 2 {
			return fmt.Errorf("usage: viper scenario apply <name>")
		}
		if err := application.ApplyScenario(args[1]); err != nil {
			return err
		}
		fmt.Println("scene applied:", args[1])
		return nil
	case "delete", "remove":
		if len(args) < 2 {
			return fmt.Errorf("usage: viper scenario delete <name>")
		}
		if err := application.DeleteScenario(args[1]); err != nil {
			return err
		}
		fmt.Println("scene deleted:", args[1])
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
		fmt.Printf("scene %q is bound to the profile %s\n", args[2], mode)
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
		fmt.Println("the profile", mode, "has no scene")
		return nil
	default:
		return fmt.Errorf("unknown scenario command %q", args[0])
	}
}

// listScenarios shows the saved scenes and the bindings.
func listScenarios(application *app.App) error {
	scenarios := application.Scenarios()
	if len(scenarios) == 0 {
		fmt.Println("  no scene is saved")
	} else {
		for name, scenario := range scenarios {
			fmt.Printf("  %-16s lighting %s at %d%%, fans %s\n", name,
				hardware.LightingEffect(scenario.Lighting.Effect),
				scenario.Lighting.Brightness,
				hardware.FanMode(scenario.Fans.CPUMode))
		}
	}

	bindings := application.Bindings()
	if len(bindings) == 0 {
		return nil
	}
	fmt.Println()
	fmt.Println("  bound scenes:")
	for profile, name := range bindings {
		fmt.Printf("    %-12s -> %s\n", profile, name)
	}
	return nil
}

// runTelemetry shows the state of the graphics card.
//
// The numbers come from the driver of the card, so this command works
// without the rights of an administrator.
func runTelemetry() error {
	application, err := app.New()
	if err != nil {
		return err
	}
	defer application.Close()

	cards, err := application.GPUTelemetry()
	if err != nil {
		return err
	}
	if len(cards) == 0 {
		fmt.Println("  no graphics card offers telemetry")
		return nil
	}
	for _, card := range cards {
		printCard(card)
	}
	return nil
}

// runPowerStatus prints the power supply of the machine.
//
// The information comes from the kernel of Windows, so this command works
// without the rights of an administrator.
func runPowerStatus() error {
	application, err := app.New()
	if err != nil {
		return err
	}
	defer application.Close()

	power, err := application.PowerStatus()
	if err != nil {
		return err
	}
	fmt.Println("  Power supply      :", power)
	if power.OnBattery {
		fmt.Println("  Battery           :", power.BatteryPercent, "%")
	}
	if power.BatteryLow {
		fmt.Println("  The battery is running low")
	}
	if power.BatteryCritical {
		fmt.Println("  The battery is almost empty")
	}
	return nil
}

// runFan shows or changes the fan configuration.
func runFan(args []string) error {
	application, err := app.New()
	if err != nil {
		return err
	}
	defer application.Close()

	if len(args) == 0 {
		state, err := application.Fans()
		if err != nil {
			return err
		}
		fmt.Println("  CPU fan    :", describeFan(state.CPUMode, state.CPUPercent))
		fmt.Println("  GPU fan    :", describeGPUFan(state))
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
		if err := application.SetCoolBoost(enabled); err != nil {
			return err
		}
		fmt.Println("CoolBoost:", onOff(enabled))
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

// applyFan applies a fan configuration and reports the result.
func applyFan(application *app.App, config hardware.FanConfig) error {
	if err := application.SetFans(config); err != nil {
		return err
	}
	state, err := application.Fans()
	if err != nil {
		return err
	}
	fmt.Println("  CPU fan    :", describeFan(state.CPUMode, state.CPUPercent))
	fmt.Println("  GPU fan    :", describeGPUFan(state))
	return nil
}

// describeFan renders a fan behaviour. The duty cycle is only meaningful
// in custom mode, in automatic mode the controller runs its own curve.
func describeFan(mode hardware.FanMode, percent int) string {
	if mode == hardware.FanCustom {
		return fmt.Sprintf("%s (%d%%)", mode, percent)
	}
	return mode.String()
}

// describeGPUFan renders the mode of the group of the graphics fan.
//
// The firmware reports the group of the processor only, so the number of
// the graphics fan is the one Viper wrote last. The flag of the state
// tells the caller that this is not a reading, the line says so as well.
//
// 固件只报告处理器这一组，显卡这一组读不回来，所以这里显示的是 Viper
// 最后一次写入的值，标记为不是从固件读来的。
func describeGPUFan(state app.FanStatus) string {
	text := describeFan(state.GPUMode, state.GPUPercent)
	if !state.GPUModeReadable {
		text += " (not readable from the firmware, this is the last value Viper wrote)"
	}
	return text
}

// runPower shows or changes the performance profile.
func runPower(args []string) error {
	application, err := app.New()
	if err != nil {
		return err
	}
	defer application.Close()

	if len(args) == 0 {
		mode, err := application.PowerMode()
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
	if err := application.SetPowerMode(mode); err != nil {
		return err
	}
	current, err := application.PowerMode()
	if err != nil {
		return err
	}
	fmt.Println("  performance profile:", current)
	return nil
}

// runLighting shows or changes the keyboard lighting.
func runLighting(args []string) error {
	application, err := app.New()
	if err != nil {
		return err
	}
	defer application.Close()

	if len(args) > 0 && args[0] == "reset" {
		if err := application.ResetLighting(); err != nil {
			return err
		}
		fmt.Println("lighting reset to the default")
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
		fmt.Println("backlight timeout:", onOff(enabled))
		return nil
	}

	if len(args) == 0 || args[0] != "set" {
		return showLighting(application)
	}

	options, err := parseLightingOptions(args[1:])
	if err != nil {
		return err
	}
	if err := application.SetLighting(*options); err != nil {
		return err
	}
	fmt.Println("lighting applied")
	return nil
}

// showLighting prints the keyboard lighting of the machine.
//
// The firmware reports the brightness only, the effect and the colours
// live in the settings of Viper. When the firmware cannot be reached the
// brightness is reported as unknown and the rest is still shown, because a
// machine whose firmware is quiet still has a lighting.
//
// 固件只报告亮度，灯效和颜色存在 Viper 自己的设置里。固件读不到的时候
// 亮度显示为未知，其余的部分照常显示。
func showLighting(application *app.App) error {
	config, err := application.Lighting()
	if err != nil {
		fmt.Fprintln(os.Stderr, "viper:", err)
		fmt.Println("  brightness  : unknown")
		config = application.StoredLighting()
	} else {
		fmt.Printf("  brightness  : %d\n", config.Brightness)
	}
	fmt.Printf("  effect      : %s\n", config.Effect)
	fmt.Printf("  speed       : %d\n", config.Speed)
	fmt.Printf("  direction   : %d\n", config.Direction)
	for i, zone := range config.Zones {
		fmt.Printf("  zone %d      : #%02X%02X%02X\n", i+1, zone.R, zone.G, zone.B)
	}
	timeout, err := application.BacklightTimeout()
	if err == nil {
		fmt.Println("  timeout     :", onOff(timeout))
	}
	if config.ColourReadable {
		fmt.Println("  note        : the colours and the effect are the ones of the machine")
	} else {
		fmt.Println("  note        : the firmware reports the brightness only, the effect and the colours come from Viper's own settings")
	}
	fmt.Println("  settings    :", application.SettingsPath())
	return nil
}

// parseLightingOptions parses the arguments of "viper lighting set".
func parseLightingOptions(args []string) (*hardware.RGBConfig, error) {
	config := &hardware.RGBConfig{Effect: hardware.EffectStatic, Speed: 3, Brightness: 50}
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
			config.PerZone = len(zoneColours) > 1
		default:
			return nil, fmt.Errorf("unknown lighting option %q", args[i])
		}
	}

	for zone, colour := range zoneColours {
		config.Zones[zone-1] = colour
	}
	return config, nil
}

// parseColor parses a hex colour.
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

// runProfile shows or changes the display overdrive and the key lock.
func runProfile(args []string) error {
	application, err := app.New()
	if err != nil {
		return err
	}
	defer application.Close()

	if len(args) == 0 {
		overdrive, err := application.LCDOverdrive()
		if err != nil {
			return err
		}
		fmt.Println("  LCD overdrive :", onOff(overdrive))

		// 锁 Windows 键是固件档案里的一个位，原厂服务写不动它，所以这一项
		// 要管理员权限。读不到就把话说明白，别把整条命令都拖垮。
		locked, err := application.WinKeyLocked()
		if err != nil {
			fmt.Fprintln(os.Stderr, "viper: 读不到 Windows 键的状态（需要管理员权限）:", err)
		} else {
			fmt.Println("  Windows key   :", onOff(locked))
		}

		sticky, err := application.StickyKeys()
		if err != nil {
			return err
		}
		fmt.Println("  Sticky keys   :", onOff(sticky))
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
		if err := application.SetWinKeyLocked(locked); err != nil {
			return err
		}
		fmt.Println("Windows key:", onOff(locked))
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
		fmt.Println("sticky keys:", onOff(enabled))
		return nil
	default:
		return fmt.Errorf("unknown profile command %q", args[0])
	}
}

// runGPU shows or changes the graphics mode of the machine.
func runGPU(args []string) error {
	application, err := app.New()
	if err != nil {
		return err
	}
	defer application.Close()

	if len(args) == 0 {
		mode, err := application.GPUMode()
		if err != nil {
			return err
		}
		fmt.Println("  graphics mode:", mode)
		return nil
	}

	mode, ok := hardware.ParseGPUMode(strings.ToLower(args[0]))
	if !ok {
		return fmt.Errorf("unknown graphics mode %q", args[0])
	}
	if err := application.SetGPUMode(mode); err != nil {
		return err
	}
	fmt.Println("  graphics mode:", mode, "(takes effect after a restart)")
	return nil
}

// runBluelight shows or changes the blue light shield.
func runBluelight(args []string) error {
	application, err := app.New()
	if err != nil {
		return err
	}
	defer application.Close()

	if len(args) == 0 {
		level, err := application.BlueLightShield()
		if err != nil {
			return err
		}
		fmt.Printf("  blue light shield: %s (%d)\n", level, int(level))
		return nil
	}

	level, ok := parseBlueLight(args[0])
	if !ok {
		return fmt.Errorf("unknown blue light level %q", args[0])
	}
	if err := application.SetBlueLightShield(level); err != nil {
		return err
	}
	fmt.Println("  blue light shield:", level)
	return nil
}

// parseBlueLight parses the level of the blue light shield.
//
// The level may be given as a number or as one of the names of the
// original application.
func parseBlueLight(text string) (quickaccess.BlueLight, bool) {
	switch strings.ToLower(text) {
	case "disabled", "off", "none":
		return quickaccess.BlueLightDisabled, true
	case "low":
		return quickaccess.BlueLightLow, true
	case "medium":
		return quickaccess.BlueLightMedium, true
	case "high":
		return quickaccess.BlueLightHigh, true
	case "long", "longhour", "long hour":
		return quickaccess.BlueLightLongHour, true
	}

	number, err := strconv.Atoi(text)
	if err != nil {
		return quickaccess.BlueLightDisabled, false
	}
	switch quickaccess.BlueLight(number) {
	case quickaccess.BlueLightDisabled,
		quickaccess.BlueLightLow,
		quickaccess.BlueLightMedium,
		quickaccess.BlueLightHigh,
		quickaccess.BlueLightLongHour:
		return quickaccess.BlueLight(number), true
	default:
		return quickaccess.BlueLightDisabled, false
	}
}

// runCharge shows or changes the charging settings.
func runCharge(args []string) error {
	application, err := app.New()
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
		fmt.Printf("  charging limit  : %d%%\n", limit)
		fmt.Println("  charging over USB:", onOff(enabled))
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
		fmt.Printf("  charging limit  : %d%%\n", percent)
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
		fmt.Println("  charging over USB:", onOff(enabled))
		return nil
	default:
		return fmt.Errorf("unknown charge command %q", args[0])
	}
}

// runBootSound shows or changes the sound at the start of the machine.
func runBootSound(args []string) error {
	application, err := app.New()
	if err != nil {
		return err
	}
	defer application.Close()

	if len(args) == 0 {
		enabled, err := application.BootSound()
		if err != nil {
			return err
		}
		fmt.Println("  boot sound:", onOff(enabled))
		return nil
	}

	enabled, err := parseOnOff(args[0])
	if err != nil {
		return err
	}
	if err := application.SetBootSound(enabled); err != nil {
		return err
	}
	fmt.Println("  boot sound:", onOff(enabled))
	return nil
}

// runDiag reports the capabilities of the machine.
//
// 报告是逐项的：有几项要走固件那一侧，那一侧需要管理员权限。读不到的那几项
// 把话说明白，不要把整份报告都拖垮。
func runDiag() error {
	application, err := app.New()
	if err != nil {
		return err
	}
	defer application.Close()

	instance, err := application.Instance()
	if err != nil {
		return err
	}
	fmt.Println("  instance        :", instance)
	fmt.Println("  route           :", application.Route())

	identity, err := application.Identity()
	if err != nil {
		return err
	}
	fmt.Println("  model           :", identity.Product)
	fmt.Println("  family          :", identity.Family)
	isPredator := hardware.IsPredatorModel(identity.Product, identity.Family)
	fmt.Println("  Predator series :", onOff(isPredator))

	modes := make([]string, 0, 4)
	for _, mode := range application.AvailablePowerModes() {
		modes = append(modes, mode.String())
	}
	fmt.Println("  profiles        :", strings.Join(modes, ", "))

	mode, err := application.PowerMode()
	report("power profile", mode, err)

	state, fanErr := application.Fans()
	if fanErr != nil {
		report("fans", nil, fanErr)
	} else {
		fmt.Println("  CPU fan         :", describeFan(state.CPUMode, state.CPUPercent))
		fmt.Println("  GPU fan         :", describeFan(state.GPUMode, state.GPUPercent))
		fmt.Println("  CoolBoost       :", onOff(state.CoolBoost))
	}

	discrete, err := application.DiscreteGPU()
	report("discrete GPU", onOff(discrete), err)

	gpuOC, cpuOC, err := application.OverclockCapability()
	if err != nil {
		report("overclock", nil, err)
	} else {
		fmt.Println("  GPU overclock   :", onOff(gpuOC))
		fmt.Println("  CPU overclock   :", onOff(cpuOC))
	}

	lighting, err := application.Lighting()
	if err != nil {
		report("key lighting", nil, err)
	} else {
		fmt.Println("  key brightness  :", lighting.Brightness)
		fmt.Println("  key effect      :", lighting.Effect)
	}

	gpuMode, err := application.GPUMode()
	report("graphics mode", gpuMode, err)

	bootSound, err := application.BootSound()
	report("boot sound", onOff(bootSound), err)

	sticky, err := application.StickyKeys()
	report("sticky keys", onOff(sticky), err)

	// 这两项要固件那一侧，也就是要管理员权限。
	locked, err := application.WinKeyLocked()
	report("Windows key", onOff(locked), err)

	fmt.Println("  settings        :", application.SettingsPath())
	return nil
}

// report prints one item of the report.
//
// 读不到的时候把原因写出来，别把整份报告都拖垮。
func report(name string, value any, err error) {
	if err != nil {
		fmt.Printf("  %-16s: 读不到（%v）\n", name, err)
		return
	}
	fmt.Printf("  %-16s: %v\n", name, value)
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
