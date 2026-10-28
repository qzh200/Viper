# agentservice

`agentservice` 实现了与宏碁原厂服务 `AcerAgentService` 的底层 TCP 网络通信客户端与二进制帧协议解包封装。

## 通信协议架构

`AcerAgentService` 是宏碁游戏笔记本（Predator / Nitro 系列）在后台运行的 Windows 服务进程，监听本地 TCP 端口。客户端通过标准 TCP Socket 与之建立长连接进行指令交互。

### 连接参数

- **默认地址**：`127.0.0.1:46933`
- **默认超时**：`5 秒`（支持自定义）
- **自动重连**：当请求写入或读取断开时，客户端自动尝试重建 TCP 连接重试一次。

### 报文封包格式

客户端向服务端发送的数据帧结构如下：

| 字段 | 长度 | 编码方式 | 说明 |
| :--- | :--- | :--- | :--- |
| Magic | 4 字节 | ASCII | 固定为 `"ACER"` |
| PacketType | 4 字节 | uint32 (小端序) | 数据包操作类型 |
| Payload | 变长 | UTF-8 JSON | 请求体内容，格式为 `{"Function":"...", "Parameter":...}` |

### 响应帧解析机制

服务端应答直接返回 JSON 文本，由于原生协议未在前置头声明内容长度，客户端采用流式花括号深度计数（Bracket Depth Counter）机制：
- 跟踪字符串转义与双引号状态；
- 当首个 `{` 出现开始计数，至对应的 `}` 归零时完成单帧 JSON 数据的完整提取。

## 数据包类型 (PacketType)

| 常量 | 数值 | 说明 |
| :--- | :--- | :--- |
| `PacketInitialization` | `0` | 查询功能能力与支持状态 (`Capability`) |
| `PacketMonitorData` | `10` | 获取系统实时温度、频率、转速等遥测数据 (`MonitorData`) |
| `PacketGetUpdatedData` | `20` | 查询指定硬件功能的当前设定值 (`Get`) |
| `PacketGetConnectedDevice` | `30` | 枚举当前接入的设备列表 (`Devices`) |
| `PacketGetLightingEffect` | `40` | 查询支持的灯效预设配置 |
| `PacketSetDeviceData` | `100` | 向目标硬件功能下发写入新参数 (`Set`) |

## 支持的功能标识 (Function)

- `OPERATING_MODE`：电源/性能档位（Quiet, Balanced, Performance, Turbo, Eco）
- `GPU_MODE`：显卡直连与混合模式（Hybrid, Discrete, Auto）
- `LIGHTING`：键盘 RGB 背光颜色、模式、速度与方向
- `FAN_CONTROL`：风扇模式（Auto, Max, Custom）及 CPU/GPU 独立转速
- `WIN_KEY`：Windows 徽标键锁定开关
- `LCD_OVERDRIVE`：屏幕响应时间超速加速开关
- `BOOT_SOUND`：开机启动声音音效开关
- `BACKLIGHT_30S`：键盘背光 30 秒无操作自动熄灭开关
- `STICKY_KEY`：粘滞键开关状态
- `BATTERY_BOOST`：电池续航加速支持
- `SOUND_MODE`：音效情景模式
- `CUSTOM_BOOT_LOGO`：自定义开机 Logo
- `ADAPTOR_STATUS`：电源适配器连接状态

## 响应状态码 (Result)

- `"0"`：操作成功执行 (`ResultOK`)
- `"3"`：功能不受支持 (`ResultUnsupported` -> `ErrUnsupported`)
- `"4"`：执行失败 (`ResultFailed` -> `ErrFailed`)

## 使用示例

```go
package main

import (
	"fmt"
	"viper/internal/agentservice"
)

func main() {
	client, err := agentservice.Connect(agentservice.Options{})
	if err != nil {
		panic(err)
	}
	defer client.Close()

	// 查询性能模式
	answer, err := client.Get(agentservice.FunctionOperatingMode)
	if err != nil {
		panic(err)
	}
	fmt.Println("响应数据:", string(answer.Data))
}
```
