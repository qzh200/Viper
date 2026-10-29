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
│   ├── agcheck/                # 原厂服务与硬件读写回环校验取证工具
│   ├── fanproto/               # 风扇模式与占空比写入延迟探测工具
│   └── winkey/                 # Windows 徽标键锁定行为探测工具
├── internal/                   # 内部核心领域实现（禁止外部模块直接引用）
│   ├── app/                    # 统一业务门面与场景引擎（协调硬件、配置与扩展服务）
│   ├── machine/                # hardware.Gaming 接口的原厂服务驱动实现（含写后轮询校验）
│   ├── agentservice/           # AcerAgentService 底层 TCP 帧协议封包与流式解析
│   ├── quickaccess/            # Acer QuickAccess 客户端（TLS + WebSocket + AES 鉴权）
│   ├── hardware/               # 领域接口与数据模型定义（Gaming、PowerMode、FanMode 等）
│   ├── config/                 # 本地配置文件（%APPDATA%/Viper/config.json）原子读写存储
│   └── probe/                  # 诊断工具套件复用的协议探针与自动化回环测试组件
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

### 单元测试与代码检查

- **运行全量测试**：
  ```powershell
  go test ./internal/...
  go test ./cmd/...
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

### B. Acer QuickAccess 协议 (`internal/quickaccess`)
- **传输层**：TLS Socket，默认监听 `127.0.0.1:5141`（跳过自签名证书验证）。
- **握手流程**：
  1. 发送 HTTP `GET /` 升级为 WebSocket（协议状态码 `101 Switching Protocols`）。
  2. 使用 AES-128-ECB 对固定握手提问字符串（`question`）及密钥 `keyA` 进行加密，并作为 `PacketType: 1`（`AuthKey`）发送完成身份鉴权。
- **功能覆盖**：显卡显示切换（混合/独显直连）、关机 USB 供电阈值、系统深色模式、任务栏自动隐藏、屏幕保护程序等。

### C. 硬件抽象与领域模型 (`internal/hardware`)
- 定义统一的硬件调控顶级接口 `hardware.Gaming` 及各细粒度子接口。
- 定义风扇模式（`FanMode`）、电源模式（`PowerMode`）、显卡工作模式（`GPUMode`）、键盘灯效（`RGBConfig`）与实时监视指标（`MonitorData`）等核心领域模型。

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
   - 尽量保持标准库与零第三方重型依赖。目前仅引入官方 `golang.org/x/sys/windows` 库用于低级 Win32 调用。
