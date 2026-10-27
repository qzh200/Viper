//go:build tools

// Command verify changes every setting the hardware layer can write, reads
// it back and puts the machine back where it was.
//
// The hardware layer checks every write against the answer of the service,
// so a setting that the machine did not take is reported as an error. This
// command proves that on the real machine: it changes a setting, checks it
// and restores the old value.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"viper/internal/agentservice"
	"viper/internal/hardware"
	"viper/internal/machine"
)

// step is one setting that is changed and put back.
type step struct {
	// name is the name of the setting.
	name string
	// apply changes the setting.
	apply func() error
	// restore puts the old value back.
	restore func() error
}

// verify changes every setting and restores it.
func verify() {
	backend, err := machine.New(machine.Options{
		OpenFirmware: func() (machine.Extras, error) {
			provider, err := hardware.OpenFirmware()
			if err != nil {
				return nil, err
			}
			return provider, nil
		},
	})
	if err != nil {
		fmt.Println("打开机器失败:", err)
		os.Exit(1)
	}
	defer backend.Close()

	// 原样写回用的会话：风扇要回到服务记着的每一个数值，只靠语义化的
	// 接口做不到。
	raw, err := agentservice.Connect(agentservice.Options{})
	if err != nil {
		fmt.Println("连接原厂服务失败:", err)
		os.Exit(1)
	}
	defer raw.Close()

	originalFans, err := raw.Get(agentservice.FunctionFanControl)
	if err != nil {
		fmt.Println("读取风扇失败:", err)
		os.Exit(1)
	}
	var originalFanPayload map[string]any
	if err := json.Unmarshal(originalFans.Data, &originalFanPayload); err != nil {
		fmt.Println("风扇配置无法解析:", err)
		os.Exit(1)
	}

	lighting, err := backend.Lighting()
	if err != nil {
		fmt.Println("读取灯效失败:", err)
		os.Exit(1)
	}
	backlight, err := backend.Backlight()
	if err != nil {
		fmt.Println("读取背光失败:", err)
		os.Exit(1)
	}
	power, err := backend.PowerMode()
	if err != nil {
		fmt.Println("读取性能档位失败:", err)
		os.Exit(1)
	}
	overdrive, err := backend.LCDOverdrive()
	if err != nil {
		fmt.Println("读取屏幕超频失败:", err)
		os.Exit(1)
	}
	sound, err := backend.BootSound()
	if err != nil {
		fmt.Println("读取开机音效失败:", err)
		os.Exit(1)
	}
	winKey, err := backend.WinKeyLocked()
	if err != nil {
		// 这一项要固件那一侧，没有管理员权限就读不到。
		fmt.Println("读不到 Windows 键的状态（需要管理员权限）:", err)
	}
	fans, err := backend.Fans()
	if err != nil {
		fmt.Println("读取风扇失败:", err)
		os.Exit(1)
	}

	fmt.Printf("原状态：灯效 %v 速度 %d 方向 %d 亮度 %d 分区 %v；背光 %d 超时 %v；档位 %v；超频 %v；音效 %v；Win 键 %v；风扇 %v\n\n",
		lighting.Effect, lighting.Speed, lighting.Direction, lighting.Brightness, lighting.Zones,
		backlight.Brightness, backlight.Timeout, power, overdrive, sound, winKey, fans.CPUMode)

	// 分区颜色：四个分区写成纯蓝，再写回原来的样子。
	zone := lighting
	zone.PerZone = true
	zone.Effect = hardware.EffectStatic
	zone.Speed = 3
	zone.Direction = 0
	zone.Brightness = 50
	for i := range zone.Zones {
		zone.Zones[i] = hardware.RGBColor{B: 255}
	}

	// 全局颜色：整块键盘写成纯绿。
	global := lighting
	global.PerZone = false
	global.Effect = hardware.EffectStatic
	global.Brightness = 50
	global.Zones[0] = hardware.RGBColor{G: 255}

	steps := []step{
		{
			name: "分区颜色",
			apply: func() error {
				return backend.SetLighting(zone)
			},
			restore: func() error {
				return backend.SetLighting(lighting)
			},
		},
		{
			name: "全局颜色",
			apply: func() error {
				return backend.SetLighting(global)
			},
			restore: func() error {
				return backend.SetLighting(lighting)
			},
		},
		{
			name: "背光亮度",
			apply: func() error {
				return backend.SetBacklight(40, backlight.Timeout)
			},
			restore: func() error {
				return backend.SetBacklight(backlight.Brightness, backlight.Timeout)
			},
		},
		{
			name: "背光超时",
			apply: func() error {
				return backend.SetBacklight(backlight.Brightness, !backlight.Timeout)
			},
			restore: func() error {
				return backend.SetBacklight(backlight.Brightness, backlight.Timeout)
			},
		},
		{
			name: "屏幕超频",
			apply: func() error {
				return backend.SetLCDOverdrive(!overdrive)
			},
			restore: func() error {
				return backend.SetLCDOverdrive(overdrive)
			},
		},
		{
			name: "开机音效",
			apply: func() error {
				return backend.SetBootSound(!sound)
			},
			restore: func() error {
				return backend.SetBootSound(sound)
			},
		},
		{
			name: "性能档位",
			apply: func() error {
				return backend.SetPowerMode(hardware.PowerPerformance)
			},
			restore: func() error {
				return backend.SetPowerMode(power)
			},
		},
		{
			name: "Windows 键",
			apply: func() error {
				return backend.SetWinKeyLocked(!winKey)
			},
			restore: func() error {
				return backend.SetWinKeyLocked(winKey)
			},
		},
		{
			name: "风扇全速",
			apply: func() error {
				return backend.SetFans(hardware.FanConfig{
					CPUMode: hardware.FanMax,
					GPUMode: hardware.FanMax,
				})
			},
			restore: func() error {
				// 原样写回，服务记着的占空比也回到原来的值。
				answer, err := raw.Set(agentservice.FunctionFanControl, originalFanPayload)
				if err != nil {
					return err
				}
				if err, failed := answer.Err(); failed {
					return err
				}
				return nil
			},
		},
	}

	failed := 0
	for _, item := range steps {
		applyErr := item.apply()
		restoreErr := item.restore()

		switch {
		case errors.Is(applyErr, machine.ErrNoProvider):
			// 这一项要固件那一侧，没有管理员权限就做不了，这不是失败。
			fmt.Printf("%-10s 需要管理员权限，跳过: %v\n", item.name, applyErr)
		case applyErr != nil && restoreErr != nil:
			failed++
			fmt.Printf("%-10s 写入失败: %v；还原也失败: %v\n", item.name, applyErr, restoreErr)
		case applyErr != nil:
			failed++
			fmt.Printf("%-10s 写入失败: %v（已还原）\n", item.name, applyErr)
		case restoreErr != nil:
			failed++
			fmt.Printf("%-10s 写入成功；还原失败: %v\n", item.name, restoreErr)
		default:
			fmt.Printf("%-10s 写入成功，读回核对通过，已还原\n", item.name)
		}
	}

	fmt.Println()
	after, _ := backend.Lighting()
	fmt.Printf("还原之后：灯效 %v 速度 %d 方向 %d 亮度 %d 分区 %v\n",
		after.Effect, after.Speed, after.Direction, after.Brightness, after.Zones)

	if failed > 0 {
		fmt.Printf("\n%d 项没有通过\n", failed)
		os.Exit(1)
	}
	fmt.Println("\n全部通过")
}
