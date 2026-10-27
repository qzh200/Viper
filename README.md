# Viper

Acer 游戏本硬件控制中心的 Go 重写版。目标是完全替代原厂
PredatorSense / NitroSense 的实用功能，同时不引入任何第三方程序。

## 设计约束

* **单一自包含可执行文件**。只依赖 Windows 自带的 `ole32.dll` 与
  `oleaut32.dll`，不需要 .NET 运行时、不需要 OpenRGB、不需要
  `inpoutx64.dll`，也不需要常驻的宿主进程。
* **纯 Go 的 COM 互操作**。`internal/com` 直接通过 `syscall` 调用
  OLE 接口，处理 VARIANT、BSTR、SAFEARRAY 与 vtable 分发。
* **Windows 专用**。当前仅支持 Windows。

## 进程依赖

**核心路径零进程依赖。** 温度、风扇、灯效、性能档位、开机音效、
图形显示模式、粘滞键、LCD 超频全部走 `root\wmi` 的
`AcerGamingFunction`。这个类是**固件动态提供**的：它不在 WMI 静态库
里，也没有任何 `WmiPrvSE` 宿主加载 Acer 模块，只依赖 Windows 的 WMI
子系统与管理员权限。

**只有两项依赖原厂进程。** 蓝光灯与 USB 充电属于原厂 Quick Access
服务（`AcerQAAgent.exe`，服务名 `AcerQAAgentSvis`），监听
`127.0.0.1:5141`。实测把该服务停掉后，`viper bluelight` 与
`viper charge` 立刻报「服务不可达」，而 `viper gpu`、`viper bootsound`
一切正常；服务重新启动后全部恢复。

原厂 PredatorSense 服务（`AcerAgentService.exe`，监听
`127.0.0.1:46933`）与两个命名管道都**不是** Viper 的依赖，Viper 只在
`cmd/palight` 这类探针里用到它们，用来给硬件写入做旁证。

## 已实现

| 能力 | 状态 |
| :--- | :--- |
| ACPI WMI 连接与实例解析 | ✅ |
| WMI 方法调用（Get/Set 系列） | ✅ |
| CPU / GPU / 主板温度 | ✅ |
| CPU / GPU 风扇转速 | ✅ |
| 风扇模式：自动 / 全速 / 固定占空比 | ✅ |
| CoolBoost 开关 | ✅ |
| 性能档位：静音 / 均衡 / 性能 / 加速 | ✅ |
| 键盘背光亮度与超时 | ✅ |
| RGB 灯效：颜色、效果、速度、方向 | ✅ |
| LCD 超频开关 | ✅ |
| Windows 键锁定 | ✅ |
| 粘滞键开关 | ✅ |
| GPU 显示模式（混合 / 仅独显 / 自动） | ✅ |
| 开机音效开关 | ✅ |
| 蓝光灯档位 | ✅ 读取已通，写入接口只接受 0–4 |
| USB 关机充电开关 | ✅ |
| 充电限制阈值（10 / 20 / 30 %） | ✅ |
| 键盘背光亮度与 30 秒超时 | ✅ 亮度可读回，超时可读回 |
| 超频能力探测 | ✅ |
| CPU / GPU 超频配置 | 🚧 读取已通，写入待确认 |
| 配置持久化与开机还原 | 🚧 |
| 单元测试（40 个用例，锁定全部实测结论） | ✅ |
| 图形界面 | 🚧 |
| PSSvc 兼容管道协议 | 🚧 |
| 启动标志自定义 | ❌ 本机固件不支持 |
| 启动动画与音效的细分选择 | ❌ 固件只暴露一个开机音效开关 |

## 运行

WMI 的实例枚举与硬件调用需要管理员权限：

```powershell
# 编译
go build -o bin/viper.exe ./cmd/viper

# 运行（需要管理员）
bin\viper.exe status
```

输出示例：

```
instance: AcerGamingFunction.InstanceName="ACPI\\PNP0C14\\APGe_0"

  CPU temperature   : 91 C
  CPU fan speed     : 2698 RPM
  GPU temperature   : 64 C
  GPU fan speed     : 2736 RPM
  Motherboard / VRM : 71 C
```

## 架构

```
cmd/viper          命令行入口
internal/com       COM/OLE 互操作层（VARIANT、BSTR、SAFEARRAY、vtable）
internal/wmi       WMI 客户端：实例解析、方法调用、参数类型自适应
internal/hardware  Acer 硬件后端：传感器、风扇、电源、灯效
internal/quickaccess 原厂 Quick Access 服务客户端（蓝光灯、USB 充电）
internal/config    设置持久化
internal/app       用例层
```

规矩与 OpenPredator 项目一致：**硬件层不认识界面，界面层不直接调用
WMI，中间夹着应用层。**

硬件连接是**按需打开**的。粘滞键与蓝光灯一类设置不需要 WMI，
因此不需要管理员权限；只有真正访问游戏硬件时才连接 provider。

## 本机实测得到的接口约束

WMI COM 接口的 vtable 布局在不同 Windows 版本上并不一致，公开的
`wbemcli.h` 头文件在本机上是错的。以下结论均由实测确定，
`cmd/slotfind` 与 `cmd/wmiscan2` 是对应的探测工具。

| 约束 | 说明 |
| :--- | :--- |
| `CoInitializeSecurity` 的时机 | 必须在 `CoInitializeEx` 之后，否则返回 `CO_E_NOTINITIALIZED` |
| `IWbemLocator::ConnectServer` | 8 参数，命名空间在前、输出指针在后；头文件里的 9 参数版本会被拒绝 |
| `IWbemServices::ExecQuery` | 6 参数，含一个 `ppCallResult` 槽位；少传会崩在 provider 内部 |
| `IWbemServices::ExecMethod` | 7 参数；`ppContext` 与 `pFlags` 必须传 NULL，传可写指针会让 provider 写入其中并崩溃 |
| `IWbemClassObject::Get` / `Put` | 槽位 4 / 5 |
| `IWbemClassObject::GetMethod` | 槽位 19 |
| `IWbemClassObject::SpawnInstance` | 槽位 15 |
| 参数 VARIANT 类型 | provider 把 UInt32 参数声明成无符号整数，却只接受 `VT_I4` 或 `VT_BSTR`，写入时按 `I4 → UI8 → UI4 → BSTR` 依次尝试 |
| 风扇模式编码 | 写入用 `1/2/3`（自动/全速/自定义）；读回时 provider 把同一个编码乘以 256 放进第三个字节，要先右移八位再取两位 |
| 风扇分组 | provider 只报告主风扇组的行为，显卡风扇组的行为读不回来；占空比用 `GetGamingFanSpeed` 按风扇编号读（1 是 CPU，4 和 2 是显卡） |
| 灯效特效编码 | `0` 静态、`1` 呼吸、`2` 霓虹、`3` 波形、`4` 切换、`5` 缩放、`6` 流星雨、`7` 闪烁。原厂档案 `LightingProfile.ini` 里每个模式都有一个 `value` 字段，编号与官方应用列出的顺序完全一致 |
| 灯效颜色 | 每个分区一个颜色，写入时是 `0x00RRGGBB` 加分区编号，分区从 1 开始。原厂档案里的 `PerKeyColor` 用的是 `0xBBGGRR`，同一颜色两种写法 |
| 灯效速度与方向 | 原厂档案写明速度范围是 `1` 到 `9`，方向是一个开关（`0` / `1`），亮度范围 `0` 到 `100` |
| 背光缓冲区 | `SetGamingKBBacklight` 的十六字节缓冲区里，**亮度在 byte2 和 byte5**，byte8 是标志位，**byte9 必须为 1**，否则固件直接丢弃缓冲区并把亮度清零 |
| 亮度读回 | `GetGamingRgbKb(1)` 返回的第二个字节就是亮度（`0x6400` 表示 100） |
| 30 秒超时 | 由动作类 `APGeAction.SetFunction` 控制，超时秒数放在第 40 位；读回用 `GetFunction(0x00088401)`，第六个字节是 `30` 即表示开启 |
| 灯效读回 | provider **不报告颜色与特效**（`GetGamingLEDColor` 与 `GetGamingLEDBehavior` 恒返回错误码 2，`GetGamingLED` 只回一个全零数组），这部分状态由 Viper 自己保存在 `%AppData%\Viper\config.json` |
| 原厂服务的灯光状态 | `AcerAgentService` 返回的 `LIGHTING` 是它**自己的缓存档案**，不跟随固件变化，因此不能用来判断一次写入是否生效；判断写入只能看 `GetGamingRgbKb` |
| Setter 的输出参数 | 回显新值而不是状态码，所以不能拿它判断成败，改为设置后读回验证 |
| 性能档位编码 | **不是** MOF 暗示的 `0/1/2/3`。实测可用值只有 `0` 静音、`1` 均衡、`4` 性能、`5` 加速；写 `2`/`3` 会被静默钳制回均衡。`6` 是固件保留档，满载时风扇反而降到静音水平，不对用户开放。原厂服务返回的 `SUPPORT_OPERATING_MODE_CAPABILITY` 同样是 `[0,1,4,5,6]` |
| 权限 | 枚举实例与调用方法都需要管理员权限 |

## 杂项设置表（`SetGamingMiscSetting`）

固件把一堆互不相干的开关放在同一张表里，标识符在低字节、值在高字节。
只有 `0x00`–`0x0D` 之间的标识符可写，其余返回 `0x2`（未定义）。
下表的对应关系是**逐项写值再回读原厂服务**验证出来的，
`cmd/miscwatch` 是对应的快照与写入工具。

| 标识符 | 功能 | 取值 |
| :--- | :--- | :--- |
| `0x02` | GPU 显示模式 | `1` 混合（MS Hybrid）、`2` 仅独显、`3` 自动选择 / DDS |
| `0x06` | 开机音效 | `0` 关、`1` 开 |
| `0x09` | 独显状态（只读） | — |
| `0x0A` | CPU 超频能力（只读） | — |
| `0x0B` | 性能档位 | `0` 静音、`1` 均衡、`4` 性能、`5` 加速 |
| `0x01`、`0x08` | 写入后没有任何可见效果，含义未确定 | — |

写入 `0x02` 会改变图形切换方式，**重启后才生效**，与官方应用的行为一致。

## 原厂服务的协议

有几个设置并不在游戏硬件 provider 里，而是由原厂的两个服务掌管。
两个协议都已逆向，Viper 只在需要时才会去连接它们。

**PredatorSense 服务（TCP `127.0.0.1:46933`）**
负责 GPU 模式、粘滞键、开机音效、性能档位、灯效等游戏硬件设置。

```
请求： "ACER" + uint32(小端) 包标识 + JSON 负载
应答： 连续的 JSON 文档，用大括号配对切分
包标识：0 初始化、10 监视数据、20 读取更新数据、100 写入设备数据
```

**Quick Access 服务（TLS WebSocket `127.0.0.1:5141`）**
负责蓝光灯、USB 关机充电、充电阈值、暗黑模式、屏保等系统设置。

```
握手： 先做 WebSocket 升级，再发 AuthPacket（PacketType = 1），
      负载是 AES-128-ECB 加密的 {"Question":…, "Key":…}
请求： {"PacketType":2,"Version":1,"Session":…,"Command":…,"Action":"Get"|"Set","Param1":…}
应答： {"PacketType":3,"Command":…,"Action":…,"Result":{"Value":…,"Text":…}}
```

两把密钥与挑战串在原厂应用里是硬编码的，`internal/quickaccess`
按同样的方式实现。蓝光灯与 USB 充电**不在这张 WMI 杂项表里**，
实测对 `0x00`–`0x20` 全表逐项写入都不会让原厂服务的读数发生变化，
因此这两项必须走 Quick Access 服务。

另外还有两个命名管道（`\\.\pipe\predatorsense_agent_service_` 与
`\\.\pipe\predatorsense_hardware_service_`），用的是原厂自己的
treadstone 二进制帧格式（`int16 命令 + uint8 参数个数 + 每参数
int32 长度 + 原始字节`），其中 `cmd 0` 是回显参数。管道需要管理员
权限才能连接。

## 测试

```powershell
go test ./...
```

四十个用例锁定的是**实测得来的结论**，不是实现的复述：性能档位的
五个编码、风扇行为的三个编码、八种灯效的编号、图形显示模式的编号、
杂项设置的标识符、颜色字与背光缓冲区的字节布局、SMBIOS 的解析、
以及 Quick Access 的加密——最后这一条用的是原厂服务握手时返回的
真实密文，改动加密实现会立刻让用例失败。

## 已知限制

* 蓝光灯的写入接口只接受 `0`–`4` 五个预设档位，本机当前是
  「自定义」（`5`），写入后无法用同一接口还原。
* 启动标志自定义需要固件支持，本机
  `HKLM\SOFTWARE\OEM\AcerAgentService\Customize_POST_Support` 为 `0`，
  原厂服务也回答 `SUPPORT_CUSTOM_BOOT_LOGO = 0`。
* GPU 风扇组的行为读不回来，Viper 如实呈现而不伪造。
* 蓝光灯、USB 充电需要原厂的 Acer Quick Access 服务在运行；
  这些设置无法只靠 WMI 完成。
* 灯效的颜色与特效无法从固件读回，Viper 用自己的配置记住它们，
  因此与官方应用**不会自动同步**：在官方应用里改灯效，Viper 读到的
  仍是上次由 Viper 写入的那一份。
* 官方应用自己保存的灯效档案在
  `C:\ProgramData\OEM\AcerLightingService\LightingProfile\LightingProfile.ini`，
  它与固件是两条独立的路径，Viper 不去改它。

## 参考

Viper 的功能范围与硬件语义参考了开源项目
[OpenPredator](https://github.com/KRWCLASSIC/OpenPredator)（MIT），
但实现完全独立，未复制其代码。

Acer、Nitro、Predator、PredatorSense、NitroSense 与 CoolBoost 是
Acer Inc. 的商标。Viper 与 Acer Inc. 无任何关联或背书关系。
