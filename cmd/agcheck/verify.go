package main

import (
	"fmt"
	"os"

	"viper/internal/agentservice"
	"viper/internal/machine"
	"viper/internal/probe"
)

func verify() {
	backend, err := machine.New(machine.Options{})
	if err != nil {
		fmt.Fprintln(stdout, "打开机器失败:", err)
		os.Exit(1)
	}
	defer backend.Close()

	raw, err := agentservice.Connect(agentservice.Options{})
	if err != nil {
		fmt.Fprintln(stdout, "连接原厂服务失败:", err)
		os.Exit(1)
	}
	defer raw.Close()

	original, results, err := probe.Verify(backend, raw)
	if err != nil {
		fmt.Fprintln(stdout, "验证失败:", err)
		os.Exit(1)
	}

	fmt.Fprintf(stdout, "原状态：灯效 %v 速度 %d 方向 %d 亮度 %d 分区 %v；背光 %d 超时 %v；档位 %v；超频 %v；音效 %v；Win 键 %v；风扇 %v\n\n",
		original.Lighting.Effect, original.Lighting.Speed, original.Lighting.Direction,
		original.Lighting.Brightness, original.Lighting.Zones,
		original.Backlight.Brightness, original.Backlight.Timeout, original.Power,
		original.Overdrive, original.BootSound, original.WinKey, original.Fans.CPUMode)

	failed := 0
	for _, result := range results {
		switch {
		case result.Skipped:
			fmt.Fprintf(stdout, "%-10s 需要管理员权限，跳过: %v\n", result.Name, result.ApplyErr)
		case result.ApplyErr != nil && result.RestoreErr != nil:
			failed++
			fmt.Fprintf(stdout, "%-10s 写入失败: %v；还原也失败: %v\n", result.Name, result.ApplyErr, result.RestoreErr)
		case result.ApplyErr != nil:
			failed++
			fmt.Fprintf(stdout, "%-10s 写入失败: %v（已还原）\n", result.Name, result.ApplyErr)
		case result.RestoreErr != nil:
			failed++
			fmt.Fprintf(stdout, "%-10s 写入成功；还原失败: %v\n", result.Name, result.RestoreErr)
		default:
			fmt.Fprintf(stdout, "%-10s 写入成功，读回核对通过，已还原\n", result.Name)
		}
	}

	fmt.Fprintln(stdout)
	after, _ := backend.Lighting()
	fmt.Fprintf(stdout, "还原之后：灯效 %v 速度 %d 方向 %d 亮度 %d 分区 %v\n",
		after.Effect, after.Speed, after.Direction, after.Brightness, after.Zones)

	if failed > 0 {
		fmt.Fprintf(stdout, "\n%d 项没有通过\n", failed)
		os.Exit(1)
	}
	fmt.Fprintln(stdout, "\n全部通过")
}
