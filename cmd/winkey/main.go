// Command winkey finds out what the WIN_KEY setting of the original service
// means and whether it can be written at all.
//
// The service answers with a number, and the number was read as "the key is
// still usable". That reading was never checked against the machine, so
// this command writes both numbers and shows what comes back.
//
// 读写的形状在 internal/probe 里，这个工具只管把过程走一遍。
package main

import (
	"fmt"
	"io"
	"os"
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

// probeWinKey writes both numbers and shows what comes back.
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
