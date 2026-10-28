# quickaccess

`quickaccess` 实现了与宏碁 `Acer QuickAccess` 系统服务的安全加密通信客户端，用于管理显示模式、关机充电及 Windows 桌面外观等扩展功能。

## 通信协议与握手流程

QuickAccess 服务监听本地端口，通信包含 TLS、WebSocket 升级与 AES 握手三层协议体系：

1. **TLS 加密传输**：
   - 默认端口：`127.0.0.1:5141`
   - 使用 TLS 建立安全传输层连接（由于原厂采用自签名证书，客户端跳过证书校验）。
2. **WebSocket 协议升级**：
   - 发送标准 HTTP `GET /` 升级握手请求（带 `Upgrade: websocket` 与随机 `Sec-WebSocket-Key`）。
   - 服务端返回 `101 Switching Protocols` 完成握手。
3. **AES 鉴权认证握手**：
   - 客户端采用固定的 AES 密钥（`A6052DC8A6E44210`）对特定握手提问字符串（`"AcerQuickAccessFunctionalityTests_HandshakingQuestion"`）进行 AES-ECB/PKCS7 算法加密。
   - 将加密后的 Base64 字符串封装入类型为 `PacketType = 1`（`AuthKey`）的认证包发送，服务端确认后鉴权通过。
4. **WebSocket 帧通信**：
   - 数据交互通过标准 WebSocket 文本与二进制帧进行，内置 Ping/Pong 链路保活机制。

## 报文与指令格式

请求结构序列化为 JSON：
```json
{
  "PacketType": 2,
  "Version": 1,
  "Session": "<UUID>",
  "Command": "<CommandName>",
  "Action": "Get" | "Set",
  "Param1": <Value>
}
```

## 支持的控制功能

| 功能命令 | 读写操作 | 说明 |
| :--- | :--- | :--- |
| `USBCharge` | Get / Set | 关机 USB 供电截止电量百分比（限制为 10%, 20%, 30%） |
| `USBChargeSwitch` | Get / Set | 关机状态下 USB 端口对外供电开关 |
| `DisplayModeSwitch` | Get / Set | 显卡输出模式（1: Hybrid 混合, 2: Discrete 独显直连, 3: Auto 自动切换） |
| `DarkMode` | Get / Set | Windows 浅色/深色主题模式 |
| `TransparentEffect` | Get / Set | 系统窗口背景透明与亚克力视觉特效 |
| `AutoHideTaskbar` | Get / Set | 任务栏自动隐藏开关 |
| `ScreenSaver` | Get / Set | 屏幕保护程序启用开关 |
| `ScreenSaverTimeout` | Get / Set | 屏幕保护程序激活前的空闲等待秒数 |

## 使用示例

```go
package main

import (
	"fmt"
	"viper/internal/quickaccess"
)

func main() {
	client, err := quickaccess.Connect(quickaccess.Options{})
	if err != nil {
		panic(err)
	}
	defer client.Close()

	// 查询显卡显示模式
	mode, err := client.DisplayMode()
	if err != nil {
		panic(err)
	}
	fmt.Println("当前显示模式:", mode)

	// 查询关机充电阈值
	limit, err := client.USBChargeLimit()
	if err != nil {
		panic(err)
	}
	fmt.Printf("关机充电截止电量: %d%%\n", limit)
}
```
