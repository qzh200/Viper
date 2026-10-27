# internal/hardware

**补充那一侧**：直接跟机器的 ACPI WMI 提供程序说话。

```
root\wmi\AcerGamingFunction   风扇表、杂项设置表（超频能力）、背光缓冲区、灯效
root\wmi\APGeAction            动作类：锁 Windows 键
```

## 它现在的位置

**原厂程序是主要调用依赖**，这一侧只是补充：硬件层主要靠原厂服务
（`internal/machine`）与 Quick Access 服务来问机器，那几个原厂程序以系统的
身份运行，不用管理员权限，而且读得到分区颜色与灯效。

这一侧只在原厂程序**给不了**的那几项上用：

| 设置 | 为什么原厂程序给不了 |
| :--- | :--- |
| 风扇表 | 原厂服务不报风扇表 |
| 超频能力 | 原厂服务自己也不实现（结果码 3） |
| 背光原始缓冲区 | 原厂服务只报亮度，不报那个十六字节的缓冲区 |
| 复位灯效 | 原厂服务能写灯效，但复位是固件的动作 |
| 锁 Windows 键 | **原厂服务收下请求却不生效**（实测：写 0 写 1 读回来一直是一个，等了五秒也不变） |

**它不是客户端路线的兜底。** 原厂服务不在的时候，程序直接报错，不偷偷换成
这一侧：直接操控固件只是补充，不是主路挂了拿固件顶上。

这一侧**需要管理员权限**：枚举实例与调用方法都要。非管理员身份下连实例都
找不到，报的是 `no AcerGamingFunction instance found`。

## 实测得到的两件事

**一、风扇表写不进去。** `GetGamingFanTable` 恒答 `0x1`（普通的表），
`SetGamingFanTable` 写了也不变。动作类的加速通道同样不响应：读回来是 `0x0`，
写了状态也不变。所以**风扇加速走原厂服务的全速模式**（实测 2722 RPM 升到
7092 RPM），不碰固件。

表号在**低字节**（1 是普通的表，3 是加速的表）。以前这里取的是高字节，所以
读出来永远是零。

**二、档案类是个常量通道。** `GetGamingProfile` 对任何查询（1、2、3、4、7、
16、132）都只答 `0x2`，`GetGamingProfileSetting` 对 1、2、3 答 1、对其余答 2，
写进去的位读回来一个都不变。用原厂服务改屏幕超频（它写进去了），固件这一
边的读数也不跟着变。所以锁 Windows 键在这台机器上走不通。

`GetGamingProfileSetting` 没有封装的原因记在 `overclock.go` 里。

## 接缝

硬件层要哪几项，由 `internal/machine` 的 `Extras` 接口说了算：

```go
// Extras is the set of abilities the program asks the firmware for.
type Extras interface {
	FanTable() (uint64, error)
	OverclockCapability() (gpu bool, cpu bool, err error)
	BacklightBuffer() ([]byte, error)
	ResetLighting() error
	WinKeyLocked() (bool, error)
	SetWinKeyLocked(locked bool) error
	Close()
}
```

## 杂项设置表（`GetGamingMiscSetting`）

固件把一堆互不相干的开关放在同一张表里，标识符在低字节、值在高字节。
这一侧现在只问一项：

| 标识符 | 功能 | 取值 |
| :--- | :--- | :--- |
| `0x0A` | CPU 超频能力（只读） | 0 表示不支持，`0xFF` 表示不知道 |

固件用低字节当状态，状态非零就表示这个设置它不认识（`0x02` 是"不知道"）。
原厂服务包了表里其余的几项（显卡模式 `0x02`、开机音效 `0x06`、性能档位
`0x0B`），所以这一侧不再写这张表。

## 背光缓冲区

`SetGamingKBBacklight` 的十六字节缓冲区里，**亮度在 byte2 和 byte5**，byte8
是标志位（三表示三十秒后自己熄灭），**byte9 必须为 1**，否则固件直接丢弃缓
冲区并把亮度清零。固件读回来的时候第九个字节是零，所以核对亮度时只看第二
个和第五个字节。

## 身份与电源

这一侧还有两样跟原厂程序无关、也不是"补充"的东西：机器身份（SMBIOS 的
`RSMB` 表）与电源状态（Windows 的内核接口）。它们只读，都不要管理员权限。

## 测试

```powershell
go test ./internal/hardware/...
```

用例锁的是实测结论：风扇表的低字节编码、背光缓冲区的字节布局、档位的
编号、杂项设置表的状态字节。

读这台机器自己的用例（身份、电源、粘滞键）只读不写，读不到就跳过。改
粘滞键的用例要环境变量 `VIPER_LIVE_MACHINE=1` 才跑，它会改回原来的样子。
