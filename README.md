# Viper

Viper 是专为宏碁（Acer）游戏笔记本（包括掠夺者 Predator 与暗影骑士 Nitro 系列）打造的跨平台硬件控制中心与工具套件。

Viper 采用 Go 语言开发，具备低资源开销、无额外运行时依赖的特点，支持直接通过命令行对笔记本的性能模式、风扇策略、键盘 RGB 背光、显卡输出模式以及系统环境进行高效调控。

---

## 核心功能

### 1. 状态监控与硬件遥测
- **整机运行遥测**：采集 CPU/GPU 温度、风扇实际转速（RPM）、核心频率、利用率及各核心独立主频。
- **系统指标监控**：监控主板温度、物理内存使用率/频率、有线以太网与 Wi-Fi 的即时上下行网络速率。

### 2. 电源与性能档位调控
- 支持 **Quiet（静音）**、**Balanced（平衡）**、**Performance（高性能）**、**Turbo（极速超频）** 与 **Eco（节能）** 模式。

### 3. 风扇策略与强冷加速
- **运转模式**：
  - `Auto`：交由硬件温控策略自动控制转速。
  - `Max`：风扇拉至最大转速运转。
  - `Custom`：支持对 CPU 与 GPU 风扇独立指定 0-100% 占空比。
- **CoolBoost 强冷加速**：支持一键开启或关闭硬件强冷加速。

### 4. 键盘 RGB 背光与灯效
- **动态动画效果**：常亮（Static）、呼吸（Breathing）、霓虹（Neon）、波形（Wave）、切换（Shifting）、缩放（Zoom）、流星雨（Meteor）、闪烁（Twinkling）共 8 种，与官方灯效面板一致。
- **个性化调节**：支持 4 分区独立 HEX 调色（`#RRGGBB`）、速度（1-9 连续可调）与亮度（1-5 档，5 为最亮）。
- **休眠与重置**：支持键盘背光 30 秒空闲自动熄灭开关，以及一键重置为出厂纯白常亮。

### 5. 显卡模式切换与外设开关
- **显卡显示模式**：支持在 **Hybrid（混合）**、**Discrete（独显直连）** 与 **Auto（自动切换）** 之间切换。
- **游戏外设开关**：支持锁定/解锁 Windows 徽标键、启用/停用屏幕 LCD Overdrive 响应时间加速、开机声音音效。

### 6. QuickAccess 系统协同集成
- **关机充电管理**：支持开启/关闭关机状态下的 USB 端口对外供电，并设置终止供电的电池阈值（10%、20%、30%）。
- **Windows 桌面定制**：一键切换 Windows 深色/浅色主题、开启/关闭窗口亚克力透明视觉特效、任务栏自动隐藏及屏幕保护程序超时等待时间。

### 7. 场景联动引擎 (Scenario Engine)
- 支持将当前满意的键盘背光配置与风扇转速保存为具名场景预设（Scenario）。
- 支持将场景与特定电源档位进行绑定：当切换电源性能模式时，系统自动应用关联的场景，实现不同工作负载下的软硬件状态联动。

---

## 项目组件结构

```text
Viper/
├── cmd/
│   ├── viperui/        # 原生 GPU 自绘图形控制中心 (MyGo)
│   ├── viper/          # 主命令行控制中心入口
│   ├── agcheck/        # 原厂服务协议探测与读写验证工具
│   ├── fanproto/       # 风扇控制协议响应时间取证工具
│   └── winkey/         # Windows 键锁定逻辑探测工具
├── internal/
│   ├── app/            # 核心业务协调层与场景引擎
│   ├── machine/        # 基于原厂服务的硬件控制驱动实现
│   ├── agentservice/   # AcerAgentService TCP 封包与解包协议库
│   ├── quickaccess/    # Acer QuickAccess TLS/WebSocket/AES 加密通信库
│   ├── domain/         # 硬件接口与领域数据模型定义
│   ├── config/         # 本地 JSON 配置文件管理库 (%APPDATA%/Viper)
│   └── probe/          # 诊断工具复用的协议探针与回环验证套件
├── build.ps1           # 自动化构建脚本
└── go.mod              # Go 模块定义
```

---

## 构建指南

### 环境要求
- 操作系统：Windows 10 / 11（x64）
- 编译工具：Go 1.20 或更高版本
- PowerShell 环境（执行自动化编译脚本）

### 编译所有组件
在项目根目录下运行 PowerShell 构建脚本：
```powershell
.\build.ps1
```
编译成功后，所有可执行文件将输出至 `bin/` 目录：
- `bin/viper.exe`
- `bin/agcheck.exe`
- `bin/fanproto.exe`
- `bin/winkey.exe`

也可单独编译主程序：
```powershell
go build -o bin/viper.exe ./cmd/viper
```

---

## 命令行使用手册 (`viper`)

### 1. 监控与信息诊断
```bash
# 查看实时硬件温度、风扇转速与负载
viper status

# 查看实时硬件遥测指标
viper telemetry

# 输出硬件能力、驱动状态与配置路径全景诊断报告
viper diag
```

### 2. 电源模式管理
```bash
# 查看当前性能档位
viper power

# 切换电源模式为高性能
viper power performance

# 切换为静音模式
viper power quiet
```

### 3. 风扇策略配置
```bash
# 查看风扇当前模式与转速
viper fans

# 将风扇交还给温控算法自动调度
viper fan auto

# 开启风扇全速运转
viper fan max

# 自定义 CPU 50% 与 GPU 60% 占空比
viper fan custom 50 60

# 开启或关闭 CoolBoost 强冷加速
viper fan coolboost on
viper fan coolboost off
```

### 4. 键盘 RGB 背光控制
```bash
# 查看当前背光参数
viper lighting

# 重置背光为出厂默认纯白色常亮
viper lighting reset

# 设置动态灯效速度（1-9）与亮度（1-5 档，5 为最亮）
viper lighting set speed 3
viper lighting set brightness 4

# 切换灯效动画模式（static/breathing/neon/wave/switch/scale/meteor/flash）
viper lighting set effect wave
viper lighting set effect breathing

# 设置全局颜色或指定分区颜色（1-4 分区）
viper lighting set color #FF0055
viper lighting set zone 1 #00FFCC all

# 开启或关闭键盘背光 30 秒空闲自动休眠熄灭
viper light timeout on
```

### 5. 显卡与显示输出
```bash
# 查看或切换显卡显示模式
viper displaymode
viper displaymode discrete
viper displaymode hybrid
viper displaymode auto
```

### 6. 外设与系统开关
```bash
# 锁定或解锁 Windows 徽标键
viper switches winkey on
viper switches winkey off

# 开启屏幕响应时间超速加速 (LCD Overdrive)
viper profile overdrive on

# 开关开机启动音效
viper bootsound off

# 开关系统深色模式与窗口透明效果
viper darkmode on
viper transparent on

# 开关任务栏自动隐藏
viper taskbar on

# 开启屏幕保护程序并设置超时秒数
viper screensaver on
viper screensaver timeout 300

# 配置关机 USB 供电阈值百分比
viper charge limit 20
viper charge switch on
```

### 7. 场景联动 (Scenario)
```bash
# 列出所有已保存的场景与档位绑定
viper scenario list

# 将当前的灯效与风扇状态保存为名为 "游戏模式" 的场景
viper scenario save 游戏模式

# 应用指定场景
viper scenario apply 游戏模式

# 将 "游戏模式" 场景绑定到 performance 档位（切换到高性能时自动生效）
viper scenario bind performance 游戏模式

# 解除特定档位的场景绑定
viper scenario unbind performance

# 删除不再使用的场景
viper scenario delete 游戏模式
```

---

## 诊断与取证工具套件

- **`agcheck`**：
  - `agcheck read`：读取原厂服务原始遥测与功能数据包。
  - `agcheck write`：执行基础功能写入测试。
  - `agcheck machine`：通过 `machine.Backend` 读取结构化硬件视图。
  - `agcheck verify`：对各项硬件功能执行自动化改动并立即还原初始状态，验证读写有效性。
- **`fanproto`**：指定模式与 CPU/GPU 占空比下发给服务，并在其后 2、4、6 秒追踪实际转速变化。
- **`winkey`**：测试 Windows 徽标键锁定的写入时延与读回生效过程。

---

## 配置文件说明

本地配置默认保存在当前用户的 AppData 目录下：
- 路径：`%APPDATA%\Viper\config.json`
- 存储内容：默认背光设定、风扇策略、电源档位记忆、自定义场景及电源模式与场景的绑定映射。
