# hardware

`hardware` 定义了 Viper 项目中与硬件交互相关的核心接口、领域数据结构，并提供了直接调用操作系统 Win32 API 与系统固件表（SMBIOS）的底层能力。

## 模块构成

### 1. 硬件抽象接口定义 (`gaming.go`)

- **`Gaming`**：组合了所有硬件控制接口的顶级接口契约。
- **细粒度子接口**：
  - `PowerModeReader`：读取与切换电源模式。
  - `GraphicsModeReader`：读取显卡直连模式与探测超频支持。
  - `LightingReader` / `LightingConfig` / `LightingWriter` / `LightingReset`：键盘背光亮度、灯效模式及 4 分区 RGB 设置。
  - `FansReader` / `FansWriter` / `CoolBoostReader`：风扇转速、运转模式与强冷加速。
  - `SwitchesReader`：Win 键锁定、屏幕响应时间超速与开机启动声音。
  - `ReadingsReader`：设备标识与基础传感器数据读取。

### 2. 领域数据模型 (`model.go`, `monitor.go`)

- **枚举类型**：
  - `FanMode`：`FanAuto`（自动）、`FanMax`（全速）、`FanCustom`（自定义）。
  - `PowerMode`：`PowerQuiet`（静音）、`PowerBalanced`（平衡）、`PowerPerformance`（高性能）、`PowerTurbo`（极速）、`PowerEco`（节能）。
  - `GPUMode`：`GPUModeHybrid`（混合）、`GPUModeDiscrete`（独显直连）、`GPUModeAuto`（自动切换）。
  - `LightingEffect`：静态常亮、呼吸、霓虹、波浪、切换、缩放、流星、闪烁。
- **结构体**：
  - `RGBColor` 与 `RGBConfig`：4 分区 RGB 颜色及动画控制。
  - `FanState` 与 `FanConfig`：风扇状态与目标下发配置。
  - `MonitorData`：汇集整机 CPU、GPU、内存、温度及网络实时遥测指标。

### 3. 电源与电池状态 (`power_status.go`, `power_status_type.go`)

直接通过 Windows Win32 API `kernel32.dll` 中的 `GetSystemPowerStatus` 读取：
- 是否连接交流电源适配器 (`OnAC`)。
- 是否处于电池供电 (`OnBattery`)。
- 电池剩余百分比 (`BatteryPercent`) 与充电中标记 (`Charging`)。
- **特点**：无需依赖宏碁原厂服务，无需管理员提权即可直接读取。

### 4. 固件身份与 SMBIOS 结构解析 (`smbios.go`)

通过 Windows Win32 API `GetSystemFirmwareTable` 获取底层 SMBIOS（`RSMB`）原始固件表数据：
- 解析 Type 1 结构（System Information），提取制造商（Manufacturer）、型号（Product）、家族（Family）、版本及序列号。
- 提供 `IsPredatorModel` 与 `IsPredator` 判断当前机器是否为宏碁掠夺者（Predator）系列。
- 支持枚举所有的 SMBIOS 结构单元（Type 4 处理器信息、Type 17 内存插槽信息等）。

### 5. Windows 辅助功能接口 (`windows.go`)

通过 Win32 API `SystemParametersInfoW` 及注册表（`HKCU\Control Panel\Accessibility\StickyKeys`）读取并配置系统粘滞键开关。

## 使用示例

```go
package main

import (
	"fmt"
	"viper/internal/hardware"
)

func main() {
	// 查询电源适配器与电池状态（免提权、免原厂服务）
	power, err := hardware.SystemPowerStatus()
	if err == nil {
		fmt.Printf("供电状态: %s (电量 %d%%)\n", power, power.BatteryPercent)
	}

	// 读取 SMBIOS 机器型号
	info, err := hardware.SystemInfo()
	if err == nil {
		fmt.Printf("机器型号: %s (%s)\n", info.Product, info.Family)
		fmt.Printf("是否属于掠夺者系列: %v\n", hardware.IsPredator())
	}
}
```
