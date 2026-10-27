# internal/quickaccess

原厂 Quick Access 服务（`AcerQAAgent.exe`，服务名 `AcerQAAgentSvis`）的
客户端。它监听 `127.0.0.1:5141`，用 TLS 承载 WebSocket。

```
握手： 先做 WebSocket 升级，再发 AuthPacket（PacketType = 1），
      负载是 AES-128-ECB 加密的 {"Question":…,"Key":…}
请求： {"PacketType":2,"Version":1,"Session":…,"Command":…,"Action":"Get"|"Set","Param1":…}
应答： {"PacketType":3,"Command":…,"Action":…,"Result":{"Value":…,"Text":…}}
```

两把密钥与挑战串在原厂应用里是硬编码的，本包按同样的方式实现。

## 本机支持的命令

下表的「服务答话」一列是**本机实测**的应答形状，方法按这个形状来解析：

| 命令 | 服务答话 | 方法 |
| :--- | :--- | :--- |
| `USBCharge` | `{"Value":30,"Text":…}` | `USBChargeLimit` / `SetUSBChargeLimit`（10、20、30） |
| `USBChargeSwitch` | `{"Value":true}` | `USBChargeEnabled` / `SetUSBChargeEnabled` |
| `BluelightShield` | `{"Value":5,"Text":…}` | `BlueLightShield` / `SetBlueLightShield`（0–4） |
| `DisplayModeSwitch` | `{"Value":2,"Description":…}` | `DisplayMode` / `SetDisplayMode`（1、2、3） |
| `DarkMode` | `{"Value":false,"Text":…}` | `DarkMode` / `SetDarkMode` |
| `TransparentEffect` | `{"Value":1,"Text":…}`（数字） | `TransparentEffect` / `SetTransparentEffect` |
| `AutoHideTaskbar` | `{"Value":0,"Text":…}`（数字） | `AutoHideTaskbar` / `SetAutoHideTaskbar` |
| `ScreenSaver` | `{"Value":false,"Text":…}`（开关） | `ScreenSaver` / `SetScreenSaver` |
| `ScreenSaverTimeout` | `{"Value":0,"Text":…}`（秒） | `ScreenSaverTimeout` / `SetScreenSaverTimeout` |

透明效果与任务栏自动隐藏，服务报的是数字（1 开、0 关），所以写回去的也是
数字；屏保服务报的是开关，写回去的也是开关。`switchValue` 两种形状都能读。

本机服务认识的另外两个命令没有做：

* `SystemUsageOSD` —— 服务**不回答**这个命令（实测会等到超时）。
* `HDR` —— 服务答得上（`{"Value":false,"Description":…}`），但原厂应用
  没有用它，先不做。

## 等待的规矩

服务不认识的命令**根本不回答**。所以每一次读取与写入都有期限，期限到了就
报错，不会把调用方永远挂在那里（`client.deadline()`）。服务还会夹带自己
的通知，读取时会跳过通知，直到属于这个命令的应答出现。

## 测试

```powershell
go test ./internal/quickaccess/...
```

`server_test.go` 里的替身服务说同一套协议，所以整个会话（握手、命令、
应答）都能在没有原厂服务的情况下跑完。

`live_probe_test.go` 连的是本机真正在跑的原厂服务，**只读**。它平时跳过，
设了环境变量 `VIPER_LIVE_QUICKACCESS=1` 才会跑，用来核对命令名与应答形状：

```powershell
$env:VIPER_LIVE_QUICKACCESS = 1
go test ./internal/quickaccess/ -run TestLiveProbe -v
```
