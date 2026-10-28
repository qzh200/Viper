// winkey 用于向 AcerAgentService 写入并验证 Windows 徽标键锁定的生效时延与行为响应。
package main

import (
	"fmt"
	"io"
	"os"
	"time"

	"viper/internal/agentservice"
	"viper/internal/probe"
)

var wait = time.Sleep

var stdout io.Writer = os.Stdout

func main() {
	client, err := agentservice.Connect(agentservice.Options{})
	if err != nil {
		fmt.Fprintln(stdout, "连接原厂服务失败:", err)
		os.Exit(1)
	}
	defer client.Close()

	if err := probeWinKey(client); err != nil {
		fmt.Fprintln(stdout, err)
		os.Exit(1)
	}
}

func probeWinKey(client probe.Service) error {
	original, err := probe.ReadWinKey(client)
	if err != nil {
		return fmt.Errorf("读取失败: %w", err)
	}
	fmt.Fprintf(stdout, "原始状态: %v\n", original.Status)

	for _, value := range []int{0, 1, 0, 1} {
		fmt.Fprintf(stdout, "\n写入 status=%d:\n", value)
		if err := probe.WriteWinKey(client, value); err != nil {
			fmt.Fprintln(stdout, "  写入失败:", err)
			continue
		}
		fmt.Fprintln(stdout, "  服务接受了这次写入")

		for i := 0; i < 5; i++ {
			wait(300 * time.Millisecond)
			status, err := probe.ReadWinKey(client)
			if err != nil {
				fmt.Fprintln(stdout, "  读取失败:", err)
				continue
			}
			fmt.Fprintf(stdout, "  %d.%d 秒后读回: %v\n", i+1, 3, status.Status)
		}
	}

	fmt.Fprintf(stdout, "\n还原成原始状态 %v\n", original.Status)
	if err := probe.WriteWinKey(client, original.Status); err != nil {
		return fmt.Errorf("写入失败: %w", err)
	}
	status, err := probe.ReadWinKey(client)
	if err != nil {
		return fmt.Errorf("读取失败: %w", err)
	}
	fmt.Fprintf(stdout, "  还原后读回: %v\n", status.Status)
	return nil
}
