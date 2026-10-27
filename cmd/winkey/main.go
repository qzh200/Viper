//go:build tools

// Command winkey finds out what the WIN_KEY setting of the original service
// means and whether it can be written at all.
//
// The service answers with a number, and the number was read as "the key is
// still usable". That reading was never checked against the machine, so
// this command writes both numbers and shows what comes back.
package main

import (
	"fmt"
	"os"
	"time"

	"viper/internal/agentservice"
)

func main() {
	client, err := agentservice.Connect(agentservice.Options{})
	if err != nil {
		fmt.Println("连接原厂服务失败:", err)
		os.Exit(1)
	}
	defer client.Close()

	original := read(client)
	fmt.Printf("原始状态: %v\n", original)

	for _, value := range []int{0, 1, 0, 1} {
		fmt.Printf("\n写入 status=%d:\n", value)
		answer, err := client.Set(agentservice.FunctionWinKey, map[string]any{"status": value})
		if err != nil {
			fmt.Println("  写入失败:", err)
			continue
		}
		if err, failed := answer.Err(); failed {
			fmt.Println("  服务拒绝了这次写入:", err)
			continue
		}
		fmt.Println("  服务接受了这次写入")

		for i := 0; i < 5; i++ {
			time.Sleep(300 * time.Millisecond)
			fmt.Printf("  %d.%d 秒后读回: %v\n", i+1, 3, read(client))
		}
	}

	fmt.Printf("\n还原成原始状态 %v\n", original)
	answer, err := client.Set(agentservice.FunctionWinKey, map[string]any{"status": original})
	if err != nil {
		fmt.Println("  写入失败:", err)
		os.Exit(1)
	}
	if err, failed := answer.Err(); failed {
		fmt.Println("  服务拒绝了这次写入:", err)
		os.Exit(1)
	}
	fmt.Printf("  还原后读回: %v\n", read(client))
}

// read reads the state of the Windows key.
func read(client *agentservice.Client) int {
	answer, err := client.Get(agentservice.FunctionWinKey)
	if err != nil {
		fmt.Println("  读取失败:", err)
		return -1
	}

	var data struct {
		Status int `json:"status"`
	}
	if err := answer.Decode(&data); err != nil {
		fmt.Println("  答案无法解析:", err)
		return -1
	}
	return data.Status
}
