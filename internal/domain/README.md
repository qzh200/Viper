# domain

`domain` 定义了 Viper 项目中核心领域接口与数据模型。

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
