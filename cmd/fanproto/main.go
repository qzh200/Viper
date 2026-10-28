// Command fanproto finds out how the original AcerAgentService reads the
// fan configuration.
//
// The service answers FAN_CONTROL with a mode and a list of fans, every fan
// carries a flag that says whether the embedded controller runs its own curve
// and a duty cycle for the case where it does not. Which of the two fields
// carries the meaning of the modes of the official application is not
// written down anywhere, so it is measured.
//
// The tool writes, waits, reads back and shows the fan speed before and
// after, so that the effect of a write is visible.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"

	"viper/internal/agentservice"
)

func main() {
	if len(os.Args) < 6 {
		fmt.Println("用法: fanproto <mode> <cpuAuto> <cpuSpeed> <gpuAuto> <gpuSpeed>")
		os.Exit(1)
	}

	mode, _ := strconv.Atoi(os.Args[1])
	cpuAuto, _ := strconv.Atoi(os.Args[2])
	cpuSpeed, _ := strconv.Atoi(os.Args[3])
	gpuAuto, _ := strconv.Atoi(os.Args[4])
	gpuSpeed, _ := strconv.Atoi(os.Args[5])

	client, err := agentservice.Connect(agentservice.Options{})
	if err != nil {
		fmt.Println("连接原厂服务失败:", err)
		os.Exit(1)
	}
	defer client.Close()

	fmt.Println("写入之前:")
	show(client)

	payload := map[string]any{
		"mode": mode,
		"custom_fan_data": []any{
			map[string]any{
				"fan_custom_auto":  cpuAuto,
				"fan_custom_speed": cpuSpeed,
				"fan_index":        0,
				"fan_name":         "CPU",
			},
			map[string]any{
				"fan_custom_auto":  gpuAuto,
				"fan_custom_speed": gpuSpeed,
				"fan_index":        0,
				"fan_name":         "GPU",
			},
		},
	}

	answer, err := client.Set(agentservice.FunctionFanControl, payload)
	if err != nil {
		fmt.Println("写入失败:", err)
		os.Exit(1)
	}
	if err, failed := answer.Err(); failed {
		fmt.Println("写入被拒绝:", err)
		os.Exit(1)
	}
	fmt.Println("写入成功")

	for i := 0; i < 3; i++ {
		time.Sleep(2 * time.Second)
		fmt.Printf("写入后 %d 秒:\n", (i+1)*2)
		show(client)
	}
}

// show shows the fan configuration and the speed of the fans.
func show(client *agentservice.Client) {
	answer, err := client.Get(agentservice.FunctionFanControl)
	if err != nil {
		fmt.Println("  读取风扇失败:", err)
		return
	}
	fmt.Println("  风扇:", string(answer.Data))

	monitor, err := client.MonitorData()
	if err != nil {
		fmt.Println("  读取监视数据失败:", err)
		return
	}
	var data map[string]any
	if err := json.Unmarshal(monitor.Data, &data); err != nil {
		fmt.Println("  监视数据无法解析:", err)
		return
	}
	fmt.Printf("  CPU 风扇 %v RPM, GPU 风扇 %v RPM, CPU %v 度\n",
		data["CPU_FANSPEED"], data["GPU1_FANSPEED"], data["CPU_TEMPERATURE"])
}
