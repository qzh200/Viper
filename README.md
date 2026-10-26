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
| 超频能力探测 | ✅ |
| CPU / GPU 超频配置 | 🚧 读取已通，写入待确认 |
| 配置持久化与开机还原 | 🚧 |
| 图形界面 | 🚧 |
| PSSvc 兼容管道协议 | 🚧 |

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
```

规矩与 OpenPredator 项目一致：**硬件层不认识界面，界面层不直接调用
WMI，中间夹着应用层。**

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
| 灯效特效编码 | `0` 静态、`1` 呼吸、`2` 霓虹、`3` 波形、`4` 切换、`5` 缩放、`6` 流星雨、`7` 闪烁，与官方应用列出的顺序一致 |
| 灯效颜色 | 每个分区一个颜色，写入时是 `0x00RRGGBB` 加分区编号，分区从 1 开始 |
| 灯效读回 | provider 只报告亮度（`GetGamingRgbKb`，范围 0 到 100），不报告颜色与特效；这部分状态由 Viper 自己保存在 `%AppData%\Viper\config.json` |
| Setter 的输出参数 | 回显新值而不是状态码，所以不能拿它判断成败，改为设置后读回验证 |
| 性能档位编码 | **不是** MOF 暗示的 `0/1/2/3`。实测可用值只有 `0` 静音、`1` 均衡、`4` 性能、`5` 加速；写 `2`/`3` 会被静默钳制回均衡。`6` 是固件保留档，满载时风扇反而降到静音水平，不对用户开放 |
| Setter 的输出参数 | 回显新值而不是状态码，所以不能拿它判断成败，改为设置后读回验证 |
| 权限 | 枚举实例与调用方法都需要管理员权限 |

## 参考

Viper 的功能范围与硬件语义参考了开源项目
[OpenPredator](https://github.com/KRWCLASSIC/OpenPredator)（MIT），
但实现完全独立，未复制其代码。

Acer、Nitro、Predator、PredatorSense、NitroSense 与 CoolBoost 是
Acer Inc. 的商标。Viper 与 Acer Inc. 无任何关联或背书关系。
