# app

`app` 是 Viper 应用程序的核心业务协调层，负责组装底层硬件后端（`machine`）、操作系统服务（`quickaccess`）与配置存储（`config`），提供高层次、统一的业务 API。

## 架构职责

1. **统一门面 (Facade Pattern)**：上层命令行或界面组件仅需与 `*app.App` 实例交互，屏蔽底层多服务、多通道的复杂性。
2. **状态持久化协同**：硬件设置变更（如背光颜色、风扇策略、电源档位）在成功写入硬件的同时，自动持久化至本地用户配置文件。
3. **场景联动引擎 (Scenario Engine)**：支持将键盘灯效和风扇策略保存为自定义命名场景，并将场景与特定电源档位进行动态绑定；当切换电源模式时，自动联动应用对应的场景预设。
4. **延迟连接与依赖注入**：硬件后端与 QuickAccess 服务采用按需建立连接机制，并支持在单元测试中注入 Fake 实例。

## 子模块功能

- **`app.go`**：应用上下文管理、服务连接生命周期维护及通信路由信息展示。
- **`power.go`**：电源性能模式切换、读取、持久化以及场景绑定自动触发。
- **`fans.go`**：CPU 与 GPU 风扇模式（自动/全速/自定义）、转速占空比及 CoolBoost 强冷开关。
- **`lighting.go`**：键盘 RGB 灯效读取与配置、4 分区独立调色、亮度调节、30 秒自动休眠开关与一键恢复出厂白色背光。
- **`scenarios.go`**：命名场景管理（创建、保存、列举、应用、删除）以及电源档位与场景的绑定/解绑。
- **`switches.go`**：外设与功能开关控制，包括 Windows 徽标键锁定、屏幕 LCD Overdrive 响应时间超速及开机声音。
- **`quickaccess.go`**：与 QuickAccess 服务协同，管理关机 USB 供电阈值、显卡显示模式、Windows 深色主题、窗口透明效果、任务栏隐藏及屏幕保护程序。
- **`readings.go`**：统一遥测与状态采集，提供基于原厂服务的整机硬件监控数据。

## 使用示例

```go
package main

import (
	"fmt"
	"viper/internal/app"
	"viper/internal/domain"
)

func main() {
	application, err := app.New()
	if err != nil {
		panic(err)
	}
	defer application.Close()

	// 1. 读取整机传感器数据
	sensors, err := application.Sensors()
	if err == nil {
		fmt.Printf("CPU: %d°C (%d RPM) | GPU: %d°C (%d RPM)\n",
			sensors.CPUTempC, sensors.CPUFanRPM, sensors.GPUTempC, sensors.GPUFanRPM)
	}

	// 2. 切换到静音模式
	if err := application.SetPowerMode(domain.PowerQuiet); err != nil {
		fmt.Println("切换失败:", err)
	}

	// 3. 保存当前状态为自定义场景并绑定到静音模式
	_ = application.SaveScenario("夜间模式")
	_ = application.BindScenario(domain.PowerQuiet.String(), "夜间模式")
}
```
