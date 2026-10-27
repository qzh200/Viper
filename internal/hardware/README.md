# internal/hardware

**兜底那一侧**：直接跟机器的 ACPI WMI 提供程序说话。

```
root\wmi\AcerGamingFunction   传感器、风扇、灯效、性能档位、杂项设置表
root\wmi\APGeAction            动作类：键盘背光、CoolBoost
```

## 它现在的位置

硬件层的主路是原厂 AcerAgentService（见 `internal/machine`）。这一侧只管
原厂服务**不管**的几项：

| 设置 | 为什么原厂服务给不了 |
| :--- | :--- |
| `CoolBoost` / 风扇表 | 加速在动作类 `APGeAction` 上，服务不报 |
| 超频能力 | 服务自己也不实现（结果码 3） |
| 背光原始缓冲区 | 服务只报亮度，不报那个十六字节的缓冲区 |
| 复位灯效 | 复位是固件的动作 |
| 锁 Windows 键 | **服务收下请求却不生效**：写 0 写 1 读回来一直是一个。锁是固件档案里的一个位（`GetGamingProfile` 的第 2 号开关，锁在第 24 位） |

这一侧同时是**顶替模式**（`internal/service`）的数据源——顶替模式的前提就
是原厂服务已经停掉，客户端那条路这时不可用。

这一侧**需要管理员权限**：枚举实例与调用方法都要。非管理员身份下连实例都
找不到，报的是 `no AcerGamingFunction instance found`。

## 固件读不到的东西

这一侧读不到的东西，原厂服务都读得到：

* **分区颜色**：`GetGamingLED` 读回来的字节始终是全零，`GetGamingLEDColor`
  与 `GetGamingLEDBehavior` 恒答 2，八种灯效的结果一模一样。
* **灯效与特效**：固件不报告，灯光的状态由原厂服务或 Viper 自己记住。
* **显卡风扇组的行为**：固件只报告主风扇组（CPU）的行为。
* **每核实时频率**：要靠性能计数器，Windows 的性能数据接口在本机初始化失败。
* **显卡使用率、网络上下行、内存占用**：这一侧没有接口。

写完之后的读回校验一律用 `GetGamingRgbKb` 一类的真实读数，不看 setter 的
输出参数——它回显新值而不是状态码，拿它判断成败会一直判成成功。

## 实测得到的接口约束

WMI COM 接口的 vtable 布局在不同 Windows 版本上并不一致，公开的
`wbemcli.h` 头文件在本机上是错。以下结论均由实测确定：

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
| 背光缓冲区 | `SetGamingKBBacklight` 的十六字节缓冲区里，**亮度在 byte2 和 byte5**，byte8 是标志位，**byte9 必须为 1**，否则固件直接丢弃缓冲区并把亮度清零 |
| 30 秒超时 | 由动作类 `APGeAction.SetFunction` 控制，超时秒数放在第 40 位；读回用 `GetFunction(0x00088401)`，第六个字节是 `30` 即表示开启 |
| 性能档位编码 | **不是** MOF 暗示的 `0/1/2/3`。实测可用值只有 `0` 静音、`1` 均衡、`4` 性能、`5` 加速、`6` 节能；写 `2`/`3` 会被静默钳制回均衡 |
| 权限 | 枚举实例与调用方法都需要管理员权限 |

## 杂项设置表（`SetGamingMiscSetting`）

固件把一堆互不相干的开关放在同一张表里，标识符在低字节、值在高字节。
只有 `0x00`–`0x0D` 之间的标识符可写，其余返回 `0x2`（未定义）。

| 标识符 | 功能 | 取值 |
| :--- | :--- | :--- |
| `0x02` | GPU 显示模式 | `1` 混合（MS Hybrid）、`2` 仅独显、`3` 自动选择 / DDS |
| `0x06` | 开机音效 | `0` 关、`1` 开 |
| `0x09` | 独显状态（只读） | — |
| `0x0A` | CPU 超频能力（只读） | — |
| `0x0B` | 性能档位 | `0` 静音、`1` 均衡、`4` 性能、`5` 加速、`6` 节能 |
| `0x01`、`0x08` | 写入后没有任何可见效果，含义未确定 | — |

写入 `0x02` 会改变图形切换方式，**重启后才生效**，与官方应用的行为一致。

## 接缝

```go
// Firmware 是这一侧对外面暴露的能力，internal/machine 只认这个接口。
type Firmware interface {
	CoolBoostEnabled() (bool, error)
	CoolBoost(enabled bool) error
	FanTable() (uint64, error)
	OverclockCapability() (gpu bool, cpu bool, err error)
	BacklightBuffer() ([]byte, error)
	ResetLighting() error
	Close()
}
```

## 测试

```powershell
go test ./internal/hardware/...
```

用例锁的是实测结论：性能档位的五个编码、风扇行为的三个编码、八种灯效的
编号、杂项设置的标识符、颜色字与背光缓冲区的字节布局、SMBIOS 的解析。
