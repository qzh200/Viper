// Command gpucross compares the readings of the graphics card with the
// ones of the original software.
//
// The original service reads the card through the interface of the driver,
// Viper reads it through the management library. Both should tell the same
// story.
//
// 协议走的是 internal/agentservice 的客户端，所以两边读的是同一个服务。
package main

import (
	"fmt"
	"io"
	"os"

	"viper/internal/agentservice"
	"viper/internal/nvidia"
	"viper/internal/probe"
)

// stdout is where the tool writes. 用例把它换成缓冲区。
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

// showCards shows the cards as the management library reports them.
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

// showService shows the graphics card as the service reports it.
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
