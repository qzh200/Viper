// Command agcheck talks to the original AcerAgentService through the client
// of the program and shows what the service answers.
//
// The tool has two modes. The read mode only asks, so it never changes
// anything. The write mode writes back the values that the service reports,
// which leaves the machine as it is and proves that the writing works.
//
// 负载的形状在 internal/probe 里，这个工具只管把过程走一遍。
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"viper/internal/agentservice"
	"viper/internal/hardware"
	"viper/internal/machine"
	"viper/internal/probe"
)

// stdout is where the tool writes. 用例把它换成缓冲区。
var stdout io.Writer = os.Stdout

func main() {
	mode := "read"
	if len(os.Args) > 1 {
		mode = os.Args[1]
	}

	switch mode {
	case "read", "write":
		client, err := agentservice.Connect(agentservice.Options{})
		if err != nil {
			fmt.Fprintln(stdout, "连接原厂服务失败:", err)
			os.Exit(1)
		}
		defer client.Close()

		if mode == "read" {
			read(client)
		} else {
			write(client)
		}
	case "machine":
		backend, err := machine.New(machine.Options{})
		if err != nil {
			fmt.Fprintln(stdout, "打开机器失败:", err)
			os.Exit(1)
		}
		defer backend.Close()

		readMachine(backend)
	case "verify":
		verify()
	default:
		fmt.Fprintln(stdout, "用法: agcheck read | write | machine | verify")
		os.Exit(1)
	}
}

// machineView is the part of the machine that the tool reads.
//
// The hardware layer of the program satisfies it, and a test can put a stand
// in its place.
type machineView interface {
	probe.Machine
	// GPUMode reads the graphics switch of the machine.
	GPUMode() (hardware.GPUMode, error)
	// Close releases the connection to the machine.
	Close()
}

// readMachine reads the machine through the hardware layer of the program.
//
// 这一段走的是 internal/machine：原厂程序是主要依赖，固件那一侧只补原厂
// 程序给不了的那几项。它是这一层与真机之间的验证。
func readMachine(backend machineView) {
	fmt.Fprintln(stdout, "机器:", backend.Instance())

	if sensors, err := backend.Sensors(); err != nil {
		fmt.Fprintln(stdout, "  读取传感器失败:", err)
	} else {
		fmt.Fprintf(stdout, "  处理器 %d 度，风扇 %d RPM；主板 %d 度；显卡 %d 度，风扇 %d RPM\n",
			sensors.CPUTempC, sensors.CPUFanRPM, sensors.SystemTempC,
			sensors.GPUTempC, sensors.GPUFanRPM)
	}

	if mode, err := backend.PowerMode(); err != nil {
		fmt.Fprintln(stdout, "  读取性能档位失败:", err)
	} else {
		fmt.Fprintln(stdout, "  性能档位:", mode)
	}

	if mode, err := backend.GPUMode(); err != nil {
		fmt.Fprintln(stdout, "  读取显卡模式失败:", err)
	} else {
		fmt.Fprintln(stdout, "  显卡模式:", mode)
	}

	if lighting, err := backend.Lighting(); err != nil {
		fmt.Fprintln(stdout, "  读取灯效失败:", err)
	} else {
		fmt.Fprintf(stdout, "  灯效 %v，速度 %d，方向 %d，亮度 %d，分区颜色 %#v\n",
			lighting.Effect, lighting.Speed, lighting.Direction, lighting.Brightness, lighting.Zones)
	}

	if state, err := backend.Fans(); err != nil {
		fmt.Fprintln(stdout, "  读取风扇失败:", err)
	} else {
		fmt.Fprintf(stdout, "  处理器风扇 %v（占空比 %d），显卡风扇 %v（占空比 %d）\n",
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
			fmt.Fprintf(stdout, "  读取%s失败: %v\n", item.name, err)
			continue
		}
		fmt.Fprintf(stdout, "  %s: %v\n", item.name, value)
	}

	if backlight, err := backend.Backlight(); err != nil {
		fmt.Fprintln(stdout, "  读取背光失败:", err)
	} else {
		fmt.Fprintf(stdout, "  背光亮度 %d，超时 %v\n", backlight.Brightness, backlight.Timeout)
	}
}

// read asks the service for everything it tells.
func read(client probe.Service) {
	answer, err := client.MonitorData()
	if err != nil {
		fmt.Fprintln(stdout, "读监视数据失败:", err)
		return
	}
	fmt.Fprintln(stdout, "监视数据:")
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
			fmt.Fprintf(stdout, "%s: 读取失败: %v\n", function, err)
			continue
		}
		if err, failed := answer.Err(); failed {
			fmt.Fprintf(stdout, "%s: %v\n", function, err)
			continue
		}
		fmt.Fprintf(stdout, "%s: %s\n", function, string(answer.Data))
	}
}

// write writes back the values that the service reports.
//
// Nothing changes on the machine, the service is told what it already knows.
//
// 灯效那一项特别要小心：全局颜色与分区颜色分开写，少写一个另一个就被清掉。
func write(client probe.Service) {
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
	if reading, err := probe.ReadLighting(client); err == nil {
		if err := probe.WriteLighting(client, reading); err != nil {
			fmt.Fprintf(stdout, "%s: 写入失败: %v\n", agentservice.FunctionLighting, err)
		} else {
			fmt.Fprintf(stdout, "%s: 写入成功 (结果 0)\n", agentservice.FunctionLighting)
		}
	}

	// 风扇：写回当前的模式与占空比。
	if reading, err := probe.ReadFan(client); err == nil {
		if err := probe.WriteFans(client, reading); err != nil {
			fmt.Fprintf(stdout, "%s: 写入失败: %v\n", agentservice.FunctionFanControl, err)
		} else {
			fmt.Fprintf(stdout, "%s: 写入成功 (结果 0)\n", agentservice.FunctionFanControl)
		}
	}
}

// report shows the result of a write.
func report(function string, answer agentservice.Answer, err error) {
	if err != nil {
		fmt.Fprintf(stdout, "%s: 写入失败: %v\n", function, err)
		return
	}
	if err, failed := answer.Err(); failed {
		fmt.Fprintf(stdout, "%s: %v\n", function, err)
		return
	}
	fmt.Fprintf(stdout, "%s: 写入成功 (结果 %s)\n", function, answer.Result)
}

// printJSON shows a payload of the service.
func printJSON(data json.RawMessage) {
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		fmt.Fprintln(stdout, string(data))
		return
	}
	text, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		fmt.Fprintln(stdout, string(data))
		return
	}
	fmt.Fprintln(stdout, string(text))
}
