# AGENTS.md

本文件为自动化 Agent（AI 编码助手）以及参与 Viper 开发的工程师提供代码库导引、架构设计说明、编码准则与开发规范。

---

## 1. 项目概述与设计定位

**Viper** 是针对宏碁（Acer）游戏笔记本（包括掠夺者 Predator 与暗影骑士 Nitro 系列）的软硬件调控中心与工具套件。
- **开发语言**：Go 1.20+
- **目标平台**：Windows 10 / 11（x64），在非 Windows 平台上提供存根（Stub）以保证跨平台语法检查和基本构建。
- **设计原则**：轻量级、无庞大运行时依赖、按需连接底层服务、写入操作具备即时校验与自动回滚保护。

---

## 2. 代码库拓扑与架构分层

整个代码库采用经典的 Go 项目分层结构：

```text
Viper/
├── cmd/                        # 各可执行程序入口（main 包）
│   ├── viper/                  # 主 CLI 工具：解析命令行参数并调度应用服务
│   ├── viperui/                # 原生图形界面（MyGo 自绘，非网页）：可截屏存 PNG
│   ├── agcheck/                # 原厂服务与硬件读写回环校验取证工具
│   ├── fanproto/               # 风扇模式与占空比写入延迟探测工具
│   └── winkey/                 # Windows 徽标键锁定行为探测工具
├── internal/                   # 内部核心领域实现（禁止外部模块直接引用）
│   ├── app/                    # 统一业务门面与场景引擎（协调硬件、配置与扩展服务）
│   ├── machine/                # domain.Gaming 接口的原厂服务驱动实现（含写后轮询校验）
│   ├── agentservice/           # AcerAgentService 底层 TCP 帧协议封包与流式解析
│   ├── quickaccess/            # Acer QuickAccess 客户端（TLS + WebSocket + AES 鉴权）
│   ├── domain/                 # 领域接口与数据模型定义（Gaming、PowerMode、FanMode 等）
│   ├── config/                 # 本地配置文件（%APPDATA%/Viper/config.json）原子读写存储
│   ├── probe/                  # 诊断工具套件复用的协议探针与自动化回环测试组件
│   └── gui/                    # 图形界面层（MyGo 原生控件，不走 HTML/CSS）
│       ├── theme/              # 外观模式：亮色 / 暗色 / 自动（纯逻辑，不依赖图形库）
│       ├── model/              # 展示模型与格式化（由硬件读数转换而来，纯逻辑）
│       ├── controller/         # 界面与硬件之间唯一的关口（依赖接口，不依赖具体实现）
│       ├── widgets/            # 复用的小组件：卡片、圆表、风扇、图标、曲线
│       ├── pages/              # 页面：home 为首页
│       ├── shell/              # 窗口外壳：侧边栏、标题栏、外观切换
│       ├── capture/            # 用 MyGo 自己的通道把窗口画成 PNG
│       └── guitest/            # 测试用的假机器与假配置存储（只被测试引用）
├── build.ps1                   # 批量编译脚本
└── go.mod                      # 依赖管理
```

### 核心分层交互关系

```text
       ┌────────────────────────┐
       │     CLI (cmd/viper)    │
       └───────────┬────────────┘
                   │
                   ▼
       ┌────────────────────────┐
       │   Facade (internal/app)│
       └───┬────────────┬───────┴──────────┐
           │            │                  │
           ▼            ▼                  ▼
┌─────────────────┐ ┌──────────────┐ ┌───────────────┐
│machine.Backend  │ │quickaccess   │ │config.Store   │
│(硬件控制与校验)  │ │(关机充电/桌面)│ │(本地配置持久化)│
└────────┬────────┘ └──────────────┘ └───────────────┘
         │
         ▼
┌─────────────────┐
│agentservice     │
│(TCP:46933 原厂) │
└─────────────────┘
```

---

## 3. 构建、测试与运行命令

### 构建流程

- **批量构建全部工具到 `bin/` 目录**：
  ```powershell
  .\build.ps1
  ```
- **单独构建主程序**：
  ```powershell
  go build -o bin/viper.exe ./cmd/viper
  go build -o bin/viperui.exe ./cmd/viperui
  ```

### 运行与调试

- **测试主程序基本功能**：
  ```powershell
  .\bin\viper.exe status
  .\bin\viper.exe diag
  ```
- **自定义原厂服务连接地址**（适用于测试代理或端口转发）：
  ```powershell
  .\bin\viper.exe --service 127.0.0.1:46933 status
  ```
- **打开图形界面**（窗口由 MyGo 自己绘制，不经过网页）：
  ```powershell
  .\bin\viperui.exe
  ```
- **用 MyGo 自己的通道截取界面并存成 PNG**：
  ```powershell
  .\bin\viperui.exe --screenshot viper.png
  .\bin\viperui.exe --theme dark --screenshot viper-dark.png
  ```

### 单元测试与代码检查

- **运行全量测试**：
  ```powershell
  go test ./internal/...
  go test ./cmd/...
  ```
- **图形界面层的覆盖率**（要求 80% 以上）：
  ```powershell
  go test -cover ./internal/gui/...
  ```
- **格式化检查**：
  ```powershell
  go fmt ./...
  ```

> [!NOTE]
> 部分测试（如 `quickaccess` 与 `agentservice` 中的集成测试）会尝试绑定本地随机端口进行 Loopback 通信。在受限安全沙箱中如果禁止绑定本地端口，测试可能会产生连接拒绝错误。

---

## 4. 关键协议与底层接口指引

### A. AcerAgentService 协议 (`internal/agentservice`)
- **传输层**：TCP Socket，默认监听 `127.0.0.1:46933`。
- **报文格式**：
  - 前 4 字节：`ACER`（ASCII 魔法头）
  - 接 4 字节：`uint32`（小端序，数据包类型代码，例如 `10` 表示监控数据，`20` 表示读取，`100` 表示写入）
  - 负载部分：UTF-8 JSON 格式字符串，形如 `{"Function":"...", "Parameter":...}`。
- **流式解包**：由于服务响应未声明长度，采用花括号深度平衡（Brace Nesting Depth）算法提取完整的 JSON 响应对象。

### A2. 键盘灯效 (`LIGHTING`) 的真机约定
以下结论来自本机 Predator 的实测，改动灯效相关代码前请先对照：

- **真值来源**：`LIGHTING` 的读取只是服务自己缓存的写入值，
  硬件真正生效的亮度与灯效模式在设备列表（`GET_CONNECTED_DEVICE`）的
  `brightness` 与 `active_mode_index` 里。写后核验要以设备列表为准。
- **亮度只有 1 到 5 档**：写 0 或超出上限都会被服务钳回上限，
  所以亮度用档位表示，不要用百分比。
- **速度范围 1 到 9**，越界被钳回上限。
- **方向只有波形（`WAVE`）与切换（`SHIFTING`）有意义**，
  其余灯效方向固定为 0，官方界面也不暴露这个选项。
- **省略字段会被清成 `null`**：一次写入必须带齐
  `brightness`、`color`、`colortype`、`device`、`direction`、`duration`、
  `effect`、`speed`、`subindex`，否则没写的设置会丢。
- **分区直控写 `device: 1`**，硬件才会切到 `DIRECT` 模式按四个 LED 分色；
  读回时也要带上设备号，否则服务只回 0 号设备那份设置，看不到分区颜色。
- **`LED_id` 的起点不统一**：分区写从 0 起算，全局写从 1 起算。

### B. Acer QuickAccess 协议 (`internal/quickaccess`)
- **传输层**：TLS Socket，默认监听 `127.0.0.1:5141`（跳过自签名证书验证）。
- **握手流程**：
  1. 发送 HTTP `GET /` 升级为 WebSocket（协议状态码 `101 Switching Protocols`）。
  2. 使用 AES-128-ECB 对固定握手提问字符串（`question`）及密钥 `keyA` 进行加密，并作为 `PacketType: 1`（`AuthKey`）发送完成身份鉴权。
- **功能覆盖**：显卡显示切换（混合/独显直连）、关机 USB 供电阈值、系统深色模式、任务栏自动隐藏、屏幕保护程序等。

### C. 硬件抽象与领域模型 (`internal/domain`)
- 定义统一的硬件调控顶级接口 `domain.Gaming` 及各细粒度子接口。
- 定义风扇模式（`FanMode`）、电源模式（`PowerMode`）、显卡工作模式（`GPUMode`）、键盘灯效（`RGBConfig`）与实时监视指标（`MonitorData`）等核心领域模型。

### D. 图形界面层 (`internal/gui`, `cmd/viperui`)
- **技术路线**：使用 [MyGo](https://github.com/egoist/mygo) 的原生控件（`github.com/egoist/mygo/ui`），由 MyGo 自己在 GPU 上绘制。**不使用 HTML/CSS，也不嵌入网页**。
- **外观模式**：只有「亮色 / 暗色 / 自动」三种，存放在 `config.json` 的 `theme` 字段。
  - 「自动」跟随系统的外观；亮色与暗色由 `shell.Apply` 强制落到 `mygo.Theme.SetSource`。
  - 模式解析在 `internal/gui/theme`（纯逻辑），落盘走 `config.Store.SetTheme`（原子写）。
- **分层与解耦**（自下而上，箭头方向即依赖方向）：
  ```text
  pages/widgets ──► controller ──► (Hardware, AppearanceStore) 接口
        │               ▲
        └──► model/theme（纯数据与纯逻辑，不认识图形库）
  ```
  - `controller` 只依赖接口；`internal/app` 的 `*app.App` 与 `*config.Store` 由 `cmd/viperui` 在装配时注入，界面层因此可以脱离原厂服务单独测试。
  - `model` 与 `theme` 不导入任何图形库，逻辑可以完整覆盖。
- **线程约定**：视图函数（`func(c *ui.Context)`）在主线程运行，绝不能在里面等待硬件。
  - 读数由 `controller` 的后台协程按周期刷新成快照，视图只读快照。
  - 写入（如切换性能档位）通过 `controller.SetPowerMode` 排队，由后台协程执行，避免卡住界面。
- **布局约定**（mygo 的 flex 行为容易踩坑）：
  - 横向的一列要占满剩下的宽，用 `Grow(1).MinWidth(0)`；
    `FillHeight()` 只管高度，宽度会缩到内容那么窄，右侧留下一条空白。
  - 根容器用 `ui.Row(c).Fill().AlignItems(ui.Stretch)`，
    子列用 `Grow(1).MinWidth(0)`，滚动区用 `ui.Scroll(c).Grow(1)`。
  - Scroll 里的内容用 `FillWidth()` 占满视口宽度；要在竖向也铺满，
    就把内容直接放进有确定高度的容器里，不要放在 Scroll 里。
- **画图约定**（用 mygo 的绘制能力，而不是堆控件）：
  - 图标、圆表、风扇都在 `widgets` 里用 `ui.Painter` 与 `ui.Path` 画出来，颜色随主题走。
  - 曲线用 `mygo/ui/chart`：`chart.Plot` 配 `chart.Area` 与 `chart.Line`，坐标轴用 `chart.NewTime`。
  - **一条路径只填一次**：`ui.Path` 的多段子路径只填第一段，要多片叶子就分多次 `FillPath`。
  - **角度先绕回一圈**：用时钟毫秒数算出来的角度在 float32 里会丢掉小数，
    几片图形会挤到同一个角度上；先在 float64 里 `math.Mod` 绕回 2π 再转 float32。
  - **转速映射**：`widgets.fanTurns` 把 RPM 折算成看得清的圈数（上限 `fanMaxTurns`），
    转速越高风扇转得越快；快到看不清时叶片变淡并补一圈残影。
- **测试**：`ui.NewTester` 可以无窗口地渲染界面并模拟点击，界面层用它做行为测试。

---

## 5. 开发约束与最佳实践 (Guidelines for Agents)

作为 Agent 修改或扩展本代码库时，请严格遵守以下规则：

1. **写后核验原则 (Verify-After-Write)**：
   - 宏碁硬件服务在写入指令后可能存在微小生效时延。在 `machine` 层进行状态修改时，必须保留或使用 `poll(read)` 轮询核对机制，确保状态真正反映到硬件后再返回。
2. **跨平台构建标签完整性**：
   - 涉及 Win32 API、注册表或 Windows 专属 DLL 的文件必须在首行包含 `//go:build windows`。
   - 相应的非 Windows 存根文件必须在首行包含 `//go:build !windows`，且保持导出符号与结构完全一致。
3. **配置原子落盘 (Atomic Save)**：
   - 配置修改必须通过 `*config.Store.Update()` 或 `Save()` 进行。底层采用先写临时文件、设置权限再重命名（Rename）的原子替换流程，杜绝配置损坏。
4. **注释与文档准则**：
   - 所有导出的包（Package）、结构体（Struct）、接口（Interface）、常量（Const）及公共方法（Method）必须包含规范的 Go Doc 注释。
   - 保持代码注释精简、准确，专注于“为什么这样做”以及硬件边界限制。
5. **依赖边界**：
   - 尽量保持标准库与零第三方重型依赖。CLI 与领域层只引入官方 `golang.org/x/sys/windows` 库用于低级 Win32 调用。
   - 图形界面层引入 `github.com/egoist/mygo`（原生 UI，不含 cgo，`CGO_ENABLED=0` 也能构建）。
6. **界面层准则**：
   - 界面层不得直接调用 `internal/machine` 或 `internal/agentservice`，一切硬件访问经由 `internal/gui/controller`。
   - 展示逻辑放进 `internal/gui/model`，外观逻辑放进 `internal/gui/theme`，两者保持纯净。
   - 界面层测试覆盖率不得低于 80%。
