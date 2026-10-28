# probe

`probe` 提供了针对底层原厂服务（`AcerAgentService`）的协议探测工具、测试夹具及真机自动化校验流程。

## 模块结构

- **`probe.go`**：定义 `Service` 基础通信接口与通用的 JSON 解码辅助函数。
- **`fan.go`**：风扇控制协议原始报文的提取、构造与下发方法（`ReadFan`, `WriteFan`, `WriteFans`）。
- **`lighting.go`**：键盘灯效原始结构解析与请求构造（`ReadLighting`, `WriteLighting`）。
- **`gpu.go`**：从监控遥测原始 JSON 中提取 GPU 核心指标与相关文本行。
- **`winkey.go`**：Windows 徽标键状态的直接读写操作（`ReadWinKey`, `WriteWinKey`）。
- **`verify.go`**：硬件控制读写完整闭环验证器（`Verify`）。

## 硬件自动化校验流程 (`Verify`)

`Verify` 函数对物理机硬件进行安全的读写能力回环校验：

1. **状态快照备份**：
   在执行任何修改前，全面读取并保存当前的硬件状态（键盘分区灯效、背光亮度与超时、电源性能档位、屏幕超速、开机声音、Win键锁定状态以及风扇策略）。
2. **逐项修改与读写校验**：
   依次对每一项功能执行写入变更并读取校验：
   - 分区颜色切换测试
   - 全局单色切换测试
   - 背光亮度调节测试
   - 背光 30 秒超时开关测试
   - 屏幕 LCD Overdrive 超速开关测试
   - 开机音效开关测试
   - 电源性能档位切换测试
   - Windows 徽标键锁定切换测试
   - 风扇全速（Max）运转测试
3. **安全即时还原**：
   每一项测试在完成后立即执行 `restore` 还原逻辑，恢复备份的初始值，确保测试不会残留非预期状态。
4. **结果统计**：
   针对每一项输出 `StepResult`，明确区分成功、需管理员权限跳过（`ErrNoProvider`）、写入失败或还原失败。

## 使用示例

```go
package main

import (
	"fmt"
	"viper/internal/agentservice"
	"viper/internal/machine"
	"viper/internal/probe"
)

func main() {
	backend, _ := machine.New(machine.Options{})
	raw, _ := agentservice.Connect(agentservice.Options{})
	defer backend.Close()
	defer raw.Close()

	original, results, err := probe.Verify(backend, raw)
	if err != nil {
		panic(err)
	}

	for _, res := range results {
		fmt.Printf("测试项目: %-10s 通过: %v\n", res.Name, res.OK())
	}
}
```
