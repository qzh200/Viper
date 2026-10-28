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
//
// 负载的形状在 internal/probe 里，这个工具只管把参数送进去。
package main

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	"viper/internal/agentservice"
	"viper/internal/probe"
)

// wait is how long the tool pauses between two readings.
//
// 用例把它换成不等待的，所以跑用例不用真的等。
var wait = time.Sleep

// stdout is where the tool writes. 用例把它换成缓冲区。
var stdout io.Writer = os.Stdout

func main() {
	if len(os.Args) < 6 {
		fmt.Fprintln(stdout, "用法: fanproto <mode> <cpuAuto> <cpuSpeed> <gpuAuto> <gpuSpeed>")
		os.Exit(1)
	}

	mode, _ := strconv.Atoi(os.Args[1])
	cpuAuto, _ := strconv.Atoi(os.Args[2])
	cpuSpeed, _ := strconv.Atoi(os.Args[3])
	gpuAuto, _ := strconv.Atoi(os.Args[4])
	gpuSpeed, _ := strconv.Atoi(os.Args[5])

	client, err := agentservice.Connect(agentservice.Options{})
	if err != nil {
		fmt.Fprintln(stdout, "连接原厂服务失败:", err)
		os.Exit(1)
	}
	defer client.Close()

	if err := probeFans(client, mode,
		probe.FanData{Name: "CPU", Auto: cpuAuto, Speed: cpuSpeed, Index: 0},
		probe.FanData{Name: "GPU", Auto: gpuAuto, Speed: gpuSpeed, Index: 0}); err != nil {
		fmt.Fprintln(stdout, err)
		os.Exit(1)
	}
}

// probeFans writes the fan configuration and shows the fans before and after.
func probeFans(client probe.Service, mode int, cpu, gpu probe.FanData) error {
	fmt.Fprintln(stdout, "写入之前:")
	show(client)

	if err := probe.WriteFan(client, mode, cpu, gpu); err != nil {
		return fmt.Errorf("写入失败: %w", err)
	}
	fmt.Fprintln(stdout, "写入成功")

	for i := 0; i < 3; i++ {
		wait(2 * time.Second)
		fmt.Fprintf(stdout, "写入后 %d 秒:\n", (i+1)*2)
		show(client)
	}
	return nil
}

// show shows the fan configuration and the speed of the fans.
func show(client probe.Service) {
	reading, err := probe.ReadFan(client)
	if err != nil {
		fmt.Fprintln(stdout, "  读取风扇失败:", err)
		return
	}
	fmt.Fprintf(stdout, "  风扇: mode %d %v\n", reading.Mode, reading.Fans)

	monitor, err := client.MonitorData()
	if err != nil {
		fmt.Fprintln(stdout, "  读取监视数据失败:", err)
		return
	}
	cpuSpeed, _ := probe.MonitorValue(monitor.Data, "CPU_FANSPEED")
	gpuSpeed, _ := probe.MonitorValue(monitor.Data, "GPU1_FANSPEED")
	temperature, _ := probe.MonitorValue(monitor.Data, "CPU_TEMPERATURE")
	fmt.Fprintf(stdout, "  CPU 风扇 %v RPM, GPU 风扇 %v RPM, CPU %v 度\n",
		cpuSpeed, gpuSpeed, temperature)
}
