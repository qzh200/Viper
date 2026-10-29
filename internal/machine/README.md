# machine

`machine` 基于 `agentservice` 客户端，实现了统一的 `domain.Gaming` 硬件抽象接口，负责将宏碁笔记本的原厂硬件协议转化为结构化的领域模型。

## 核心职责

1. **硬件抽象实现**：实现 `domain.Gaming` 接口，提供电源模式、显卡直连、风扇转速、键盘灯效与外设开关的完整控制。
2. **写后校验机制 (Write-Verification)**：向硬件服务写入控制参数后，自动采用轮询重试机制（默认超时 3 秒，间隔 100 毫秒）读回硬件状态，确认配置真实生效后再返回成功。
3. **数据模型映射与容错**：负责处理各硬件功能在原厂协议中的数值编码与标准模型之间的双向映射转换。

## 支持的控制与采集能力

### 1. 硬件遥测与传感器 (`Monitor` / `Sensors`)

解析 `agentservice.PacketMonitorData` 返回的实时系统数据：
- **CPU**：封装温度、风扇转速（RPM）、利用率（%）、当前主频（MHz）、最大主频、各核心独立频率列表。
- **GPU**：核心温度、风扇转速（RPM）、利用率（%）、当前频率、最大频率、空闲状态（Idle）。
- **系统与主板**：主板温度。
- **内存**：物理内存总容量（MB）、使用率百分比、工作频率（MHz）。
- **网络**：Wi-Fi 与以太网的即时上行/下行传输速率（Kbps）。

### 2. 电源性能档位 (`PowerMode` / `SetPowerMode`)

| 模式 | 数值编码 | 说明 |
| :--- | :--- | :--- |
| `PowerQuiet` | `0` | 静音低功耗模式 |
| `PowerBalanced` | `1` | 标准平衡模式 |
| `PowerPerformance` | `4` | 高性能模式 |
| `PowerTurbo` | `5` | 极速超频狂暴模式 |
| `PowerEco` | `6` | 节能省电模式 |

### 3. 显卡切换模式 (`GPUMode` / `SetGPUMode`)

- **混合模式 (`GPUModeHybrid`)**：对应服务参数 `0`，集显与独显协同工作。
- **独显直连 (`GPUModeDiscrete`)**：对应服务参数 `1`，屏幕直接由独立 GPU 输出。
- **自动切换 (`GPUModeAuto`)**：对应服务参数 `2`，根据负载自动切换（NVIDIA Advanced Optimus / DDS）。

### 4. 风扇与强冷控制 (`Fans` / `SetFans` / `CoolBoost`)

- **运转模式**：
  - 自动 (`FanAuto`)：原厂温控算法动态调节。
  - 全速 (`FanMax`)：风扇拉至最高转速。
  - 自定义 (`FanCustom`)：针对 CPU 与 GPU 风扇分别指定 0-100% 占空比。
- **强冷模式 (`CoolBoost`)**：当 CPU 与 GPU 风扇同时进入全速状态时视为开启。

### 5. 键盘 RGB 背光 (`Lighting` / `SetLighting` / `ResetLighting`)

- **动画效果**：常亮 (`STATIC`)、呼吸 (`BREATHING`)、霓虹 (`NEON`)、波浪 (`WAVE`)、切换 (`SHIFTING`)、缩放 (`ZOOM`)、流星 (`METEOR`)、闪烁 (`TWINKLING`)。
- **速度与方向**：速度范围 1-9，方向 0 或 1。
- **分区配置**：支持 4 个独立 RGB 颜色分区（HEX 格式 `#RRGGBB`）。
- **背光休眠**：支持读取与配置键盘背光 30 秒空闲自动熄灭开关 (`BACKLIGHT_30S`)。
- **重置功能**：一键将键盘还原为全区白色常亮（亮度 100%）。

### 6. 外设开关 (`SwitchesReader`)

- **Win 键锁定 (`WinKeyLocked`)**：注意原厂服务的语义：状态 `0` 表示按键锁定（禁用），状态 `1` 表示按键激活。
- **屏幕响应时间超速 (`LCDOverdrive`)**：降低屏幕拖影响应时间。
- **开机启动音效 (`BootSound`)**：控制笔记本开机蜂鸣或问候音效。

## 使用示例

```go
package main

import (
	"fmt"
	"viper/internal/domain"
	"viper/internal/machine"
)

func main() {
	backend, err := machine.New(machine.Options{})
	if err != nil {
		panic(err)
	}
	defer backend.Close()

	// 切换电源模式为高性能并校验
	if err := backend.SetPowerMode(domain.PowerPerformance); err != nil {
		fmt.Println("切换失败:", err)
		return
	}
	fmt.Println("电源模式已生效")
}
```
