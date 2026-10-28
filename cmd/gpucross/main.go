// gpucross 用于对比直接调用 NVIDIA NVML 读取的显卡指标与 AcerAgentService 返回的监控数据。
package main

import (
	"fmt"
	"io"
	"os"

	"viper/internal/agentservice"
	"viper/internal/nvidia"
	"viper/internal/probe"
)

var stdout io.Writer = os.Stdout

func main() {
	cards, err := nvidia.Read()
	if err != nil {
		fmt.Fprintln(stdout, "nvidia:", err)
		os.Exit(1)
	}

	fmt.Fprintln(stdout, "=== Viper, management library ===")
	showCards(cards)

	version, err := nvidia.DriverVersion()
	if err == nil {
		fmt.Fprintln(stdout, "  driver        :", version)
	}

	fmt.Fprintln(stdout, "\n=== original service, interface of the driver ===")
	client, err := agentservice.Connect(agentservice.Options{})
	if err != nil {
		fmt.Fprintln(stdout, "  the service did not answer:", err)
		return
	}
	defer client.Close()

	showService(client)
}

func showCards(cards []nvidia.Telemetry) {
	for i, card := range cards {
		fmt.Fprintf(stdout, "  card %d: %s\n", i, card.Name)
		fmt.Fprintf(stdout, "    usage        : %d%% (memory %d%%)\n", card.UsagePercent, card.MemoryUsagePercent)
		fmt.Fprintf(stdout, "    graphics clock: %d MHz (max %d MHz)\n", card.GraphicsClockMHz, card.MaxGraphicsClockMHz)
		fmt.Fprintf(stdout, "    memory clock : %d MHz\n", card.MemoryClockMHz)
		fmt.Fprintf(stdout, "    memory       : %d of %d bytes\n", card.MemoryUsedBytes, card.MemoryTotalBytes)
		fmt.Fprintf(stdout, "    performance  : %d\n", card.PState)
	}
}

func showService(client probe.Service) {
	monitor, err := client.MonitorData()
	if err != nil {
		fmt.Fprintln(stdout, "  the service did not answer:", err)
		return
	}
	if err, failed := monitor.Err(); failed {
		fmt.Fprintln(stdout, "  the service did not answer:", err)
		return
	}

	for _, line := range probe.GraphicsLines(monitor.Data) {
		fmt.Fprintln(stdout, " ", line)
	}
}
