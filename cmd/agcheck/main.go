//go:build tools

// Command agcheck talks to the original AcerAgentService through the client
// of the program and shows what the service answers.
//
// The tool has two modes. The read mode only asks, so it never changes
// anything. The write mode writes back the values that the service reports,
// which leaves the machine as it is and proves that the writing works.
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"viper/internal/agentservice"
	"viper/internal/hardware"
	"viper/internal/machine"
)

func main() {
	mode := "read"
	if len(os.Args) > 1 {
		mode = os.Args[1]
	}

	client, err := agentservice.Connect(agentservice.Options{})
	if err != nil {
		fmt.Println("连接原厂服务失败:", err)
		os.Exit(1)
	}
	defer client.Close()

	switch mode {
	case "read":
		read(client)
	case "write":
		write(client)
	case "machine":
		readMachine()
	case "verify":
		verify()
	default:
		fmt.Println("用法: agcheck read | write | machine | verify")
		os.Exit(1)
	}
}

// machine reads the machine through the hardware layer of the program.
//
// 这一段走的是 internal/machine：原厂服务是主路，固件那一侧只补原厂服务
// 不管的几项。它是这一层与真机之间的验证。
func readMachine() {
	backend, err := machine.New(machine.Options{
		OpenFirmware: hardware.OpenFirmware,
	})
	if err != nil {
		fmt.Println("打开机器失败:", err)
		os.Exit(1)
	}
	defer backend.Close()

	fmt.Println("机器:", backend.Instance())

	if sensors, err := backend.Sensors(); err != nil {
		fmt.Println("  读取传感器失败:", err)
	} else {
		fmt.Printf("  处理器 %d 度，风扇 %d RPM；主板 %d 度；显卡 %d 度，风扇 %d RPM\n",
			sensors.CPUTempC, sensors.CPUFanRPM, sensors.SystemTempC,
			sensors.GPUTempC, sensors.GPUFanRPM)
	}

	if mode, err := backend.PowerMode(); err != nil {
		fmt.Println("  读取性能档位失败:", err)
	} else {
		fmt.Println("  性能档位:", mode)
	}

	if mode, err := backend.GPUMode(); err != nil {
		fmt.Println("  读取显卡模式失败:", err)
	} else {
		fmt.Println("  显卡模式:", mode)
	}

	if lighting, err := backend.Lighting(); err != nil {
		fmt.Println("  读取灯效失败:", err)
	} else {
		fmt.Printf("  灯效 %v，速度 %d，方向 %d，亮度 %d，分区颜色 %#v\n",
			lighting.Effect, lighting.Speed, lighting.Direction, lighting.Brightness, lighting.Zones)
	}

	if state, err := backend.Fans(); err != nil {
		fmt.Println("  读取风扇失败:", err)
	} else {
		fmt.Printf("  处理器风扇 %v（占空比 %d），显卡风扇 %v（占空比 %d）\n",
			state.CPUMode, state.CPUPercent, state.GPUMode, state.GPUPercent)
	}

	for _, item := range []struct {
		name string
		read func() (bool, error)
	}{
		{"Windows 键锁着", backend.WinKeyLocked},
		{"屏幕超频", backend.LCDOverdrive},
		{"开机音效", backend.BootSound},
	} {
		value, err := item.read()
		if err != nil {
			fmt.Printf("  读取%s失败: %v\n", item.name, err)
			continue
		}
		fmt.Printf("  %s: %v\n", item.name, value)
	}

	if backlight, err := backend.Backlight(); err != nil {
		fmt.Println("  读取背光失败:", err)
	} else {
		fmt.Printf("  背光亮度 %d，超时 %v\n", backlight.Brightness, backlight.Timeout)
	}

	// 原厂服务不管的几项走固件那一侧，这一层需要管理员权限。
	if _, err := backend.FanTable(); err != nil {
		fmt.Println("  读取风扇表失败:", err)
	} else {
		fmt.Println("  风扇表: 读到了")
	}
}

// read asks the service for everything it tells.
func read(client *agentservice.Client) {
	answer, err := client.MonitorData()
	if err != nil {
		fmt.Println("读监视数据失败:", err)
		return
	}
	fmt.Println("监视数据:")
	printJSON(answer.Data)

	for _, function := range []string{
		agentservice.FunctionOperatingMode,
		agentservice.FunctionGPUMode,
		agentservice.FunctionLighting,
		agentservice.FunctionFanControl,
		agentservice.FunctionWinKey,
		agentservice.FunctionLCDOverdrive,
		agentservice.FunctionBootSound,
		agentservice.FunctionBacklightTimeout,
		agentservice.FunctionStickyKey,
	} {
		answer, err := client.Get(function)
		if err != nil {
			fmt.Printf("%s: 读取失败: %v\n", function, err)
			continue
		}
		if err, failed := answer.Err(); failed {
			fmt.Printf("%s: %v\n", function, err)
			continue
		}
		fmt.Printf("%s: %s\n", function, string(answer.Data))
	}
}

// write writes back the values that the service reports.
//
// Nothing changes on the machine, the service is told what it already knows.
func write(client *agentservice.Client) {
	// 性能档位：写回当前档位。
	if answer, err := client.Get(agentservice.FunctionOperatingMode); err == nil {
		var data struct {
			Mode int `json:"mode"`
		}
		if answer.Decode(&data) == nil {
			answer, err := client.Set(agentservice.FunctionOperatingMode,
				map[string]any{"mode": data.Mode})
			report(agentservice.FunctionOperatingMode, answer, err)
		}
	}

	// 图形模式：写回当前模式。
	if answer, err := client.Get(agentservice.FunctionGPUMode); err == nil {
		var data struct {
			Mode int `json:"mode"`
		}
		if answer.Decode(&data) == nil {
			answer, err := client.Set(agentservice.FunctionGPUMode,
				map[string]any{"mode": data.Mode})
			report(agentservice.FunctionGPUMode, answer, err)
		}
	}

	// 开关类：写回当前状态。
	for _, function := range []string{
		agentservice.FunctionWinKey,
		agentservice.FunctionLCDOverdrive,
		agentservice.FunctionBootSound,
		agentservice.FunctionBacklightTimeout,
	} {
		if answer, err := client.Get(function); err == nil {
			var data struct {
				Status int `json:"status"`
			}
			if answer.Decode(&data) == nil {
				answer, err := client.Set(function, map[string]any{"status": data.Status})
				report(function, answer, err)
			}
		}
	}

	// 粘滞键：原厂服务自己也不支持这一项。
	if answer, err := client.Set(agentservice.FunctionStickyKey, map[string]any{"status": 0}); err == nil {
		report(agentservice.FunctionStickyKey, answer, nil)
	}

	// 灯效：写回当前的分区颜色、效果、速度、方向与亮度。
	if answer, err := client.Get(agentservice.FunctionLighting); err == nil {
		var data struct {
			LEDs []struct {
				LEDID int    `json:"LED_id"`
				Color string `json:"color"`
			} `json:"LEDs"`
			Brightness int    `json:"brightness"`
			Effect     string `json:"effect"`
			Speed      int    `json:"speed"`
			Direction  int    `json:"direction"`
		}
		if answer.Decode(&data) == nil {
			leds := make([]any, 0, len(data.LEDs))
			for _, led := range data.LEDs {
				leds = append(leds, map[string]any{
					"LED_id": led.LEDID,
					"color":  led.Color,
					"status": 1,
				})
			}
			answer, err := client.Set(agentservice.FunctionLighting, map[string]any{
				"LEDs":       leds,
				"brightness": data.Brightness,
				"effect":     data.Effect,
				"speed":      data.Speed,
				"direction":  data.Direction,
			})
			report(agentservice.FunctionLighting, answer, err)
		}
	}

	// 风扇：写回当前的模式与占空比。
	if answer, err := client.Get(agentservice.FunctionFanControl); err == nil {
		var data struct {
			Mode          int `json:"mode"`
			CustomFanData []struct {
				Auto  int    `json:"fan_custom_auto"`
				Speed int    `json:"fan_custom_speed"`
				Name  string `json:"fan_name"`
			} `json:"custom_fan_data"`
		}
		if answer.Decode(&data) == nil {
			fans := make([]any, 0, len(data.CustomFanData))
			for _, fan := range data.CustomFanData {
				fans = append(fans, map[string]any{
					"fan_custom_auto":  fan.Auto,
					"fan_custom_speed": fan.Speed,
					"fan_index":        0,
					"fan_name":         fan.Name,
				})
			}
			answer, err := client.Set(agentservice.FunctionFanControl, map[string]any{
				"mode":            data.Mode,
				"custom_fan_data": fans,
			})
			report(agentservice.FunctionFanControl, answer, err)
		}
	}
}

// report shows the result of a write.
func report(function string, answer agentservice.Answer, err error) {
	if err != nil {
		fmt.Printf("%s: 写入失败: %v\n", function, err)
		return
	}
	if err, failed := answer.Err(); failed {
		fmt.Printf("%s: %v\n", function, err)
		return
	}
	fmt.Printf("%s: 写入成功 (结果 %s)\n", function, answer.Result)
}

// printJSON shows a payload of the service.
func printJSON(data json.RawMessage) {
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		fmt.Println(string(data))
		return
	}
	text, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		fmt.Println(string(data))
		return
	}
	fmt.Println(string(text))
}
