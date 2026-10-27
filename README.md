# Viper

Acer 游戏本硬件控制中心的 Go 重写版。

**原厂程序是主要调用依赖。** 官方 PredatorSense 的后台服务
（`AcerAgentService`）是这台机器上真正管着灯光、风扇、性能档位与传感器读数
的那一路，Viper 作为它的客户端去问，**不需要管理员权限**。Quick Access 服务
（`AcerQAAgent`）管着蓝光灯、USB 充电这些系统设置。

**直接操控固件只是补充。** 只有原厂程序给不了的那几项——风扇表、超频能力、
背光原始缓冲区、复位灯效、锁 Windows 键——才走 ACPI 固件那一侧，那一侧需
要管理员权限。它不是主路挂了拿来顶上的备胎。

```
                 ┌────────────────────────────────────────────┐
                 │  Viper                                     │
                 │                                            │
   原厂服务 ───▶ │  internal/agentservice ─▶ internal/machine │ ──▶ 硬件层接口
 (46933, SYSTEM) │   （协议客户端）            （硬件层）      │      ↑
                 │                                            │      │
   原厂 Quick ─▶ │  internal/quickaccess（蓝光灯、USB 充电等）──┘      │
  (5141, SYSTEM) │                                            │      │
                 │  internal/hardware（补充：风扇表、超频能力、  │      │
 (root\wmi)      │    背光缓冲区、复位灯效、锁 Windows 键）──────┘      │
                 └────────────────────────────────────────────┘
```

## 现在的状态

| 部分 | 状态 |
| :--- | :--- |
| 硬件层 `internal/machine` + `internal/agentservice` | ✅ 已完成，真机验证通过 |
| 命令行接到硬件层 | ✅ 已接上，**不需要管理员权限** |
| 固件那一侧 | ✅ 只剩补充的那几项 |
| 图形界面 | 🚧 以后再说 |

命令行会报出自己走的是哪条路：

```
$ viper status
instance: AcerAgentService
route  : 原厂服务 AcerAgentService
```

原厂服务不在的时候，程序直接把实情报给用的人，**不直接改走固件**——直接操控
固件只是补充，不是这条路兜底的：

```
$ viper --service 127.0.0.1:1 status
viper: 原厂服务不可达: agentservice: 原厂服务不可达: 127.0.0.1:1: dial tcp ...
  直接操控固件只是补充（原厂程序给不了的那几项才用得上），不是这条路兜底的。
```

## 原厂程序给得到什么

固件那一侧读不到的东西，原厂服务都读得到。下表是本机实测的：

| 能力 | 固件（WMI） | 原厂服务 |
| :--- | :--- | :--- |
| 分区颜色 | ❌ 读回来永远是全零 | ✅ 四个分区的真实颜色 |
| 灯效 | ❌ 八个灯效的结果一模一样 | ✅ 效果名、速度、方向、亮度 |
| 显卡风扇组的行为 | ❌ 只报告主风扇组 | ✅ 两组都报 |
| 风扇转速与占空比 | ✅ 能写 | ✅ 能写，而且**固件的加速表在本机是死的** |
| 每核实时频率 | ❌ 性能计数器在本机初始化失败 | ✅ 32 核每一个都有 |
| 显卡使用率 | ❌ | ✅ |
| 网络上下行 | ❌ | ✅ |
| 内存占用与频率 | ❌ | ✅ |
| 睿频上限 | 2419（标称频率） | 5000（真实的睿频上限） |
| 性能档位、显卡模式、屏幕超频、开机音效、背光超时 | ✅ 能写 | ✅ 能写 |

实测这一段的时候，本机会话**没有管理员权限**：固件那一侧连实例都找不到
（`no AcerGamingFunction instance found`），原厂服务这一侧却什么都能读。
这就是换路线的最大收益——它不以管理员身份运行。

## 固件那一侧补什么

只有这五项，而且它们都要管理员权限：

| 设置 | 为什么原厂程序给不了 |
| :--- | :--- |
| 风扇表 | 原厂服务不报风扇表 |
| 超频能力 | 原厂服务自己也不实现（结果码 3） |
| 背光原始缓冲区 | 原厂服务只报亮度，不报那个十六字节的缓冲区 |
| 复位灯效 | 原厂服务能写灯效，但复位是固件的动作 |
| 锁 Windows 键 | **原厂服务收下请求却不生效**（见下） |

固件的连接是**按需打开**的，所以一台没有管理员权限的机器照样能用上原厂
程序给的那几项。连接打不开时报的是「这一项需要固件接口，原厂程序不提供，
需要管理员权限」，不会拿别的东西冒充。

## 实测得到的编码

这一节里的每一条都是在这台机器上量出来的，不是照着文档推的。

**风扇**（`FAN_CONTROL`）：顶层 `mode` 是开关，每个风扇的
`fan_custom_auto` 与 `fan_custom_speed` 是它的行为与占空比。

| 写法 | 实测 |
| :--- | :--- |
| `mode: 0` | 自动，控制器跑自己的曲线（2722 RPM） |
| `mode: 1` | 全速，**7092 RPM** |
| `mode: 2` + `fan_custom_auto: 0` + `fan_custom_speed: 50` | 固定占空比，**4400 RPM** |

Viper 写的时候照服务自己的三个模式来：两个风扇都自动写模式 0，都全速写
模式 1，其余写模式 2。写模式 2 的话，服务档案
`HKLM\SOFTWARE\OEM\AcerAgentService\FanControl\CurrentFanMode` 里记的是「自
定义」，官方应用看到的也就不是自动了。

**风扇加速**就是原厂服务的全速模式。固件那一侧也有一个加速表，但实测在本
机是**死的通道**：动作类读回来是 0，写了状态也不变，风扇表恒是 1（普通的
表）。所以这一项走原厂程序，不碰固件。

**性能档位**（`OPERATING_MODE`）：编号是 `0` 静音、`1` 均衡、`4` 性能、
`5` 加速、`6` 节能，与固件那一侧的编号一样。

**显卡模式**（`GPU_MODE`）：服务从零开始数，`0` 混合、`1` 仅独显、
`2` 自动。切换图形模式要重启才生效，与官方应用的行为一致。

**开关类**（`WIN_KEY`、`LCD_OVERDRIVE`、`BOOT_SOUND`、`BACKLIGHT_30S`）：
形状都是 `{"status": 0 或 1}`。`WIN_KEY` 的编号 1 表示**这个键还能用**，
所以锁住的时候是 0。

**灯光**（`LIGHTING`）：四个分区各一个颜色（`#RRGGBB`），加上效果名、速度、
方向与亮度。写亮度时要把四个分区一起写回去，否则原厂服务会把颜色清掉。

**锁 Windows 键**：原厂服务**收下请求**（结果码 0），但写 0 写 1 读回来一直是
一个，等了五秒也不变。固件那一侧的档案类在本机是个常量通道
（`GetGamingProfile` 对任何查询都只答 `0x2`，`SetGamingProfile` 写不进去），
所以这台机器上锁不住这个键。Viper 如实报错，不假装成功。

## 写完要读回核对

每一项写入之后都把配置读回来核对：灯光（颜色、效果、速度、方向、亮度）、
背光（亮度与超时）、性能档位、开关类（屏幕超频、开机音效）、风扇。机器没
有接受就在错误信息里说清楚是哪一个。

核对要**轮询一小会儿**（最多三秒）：有的设置是服务交给别的服务才生效，读
回来的值会晚一点才变。读一次就下结论会把还没生效的设置报成失败。

两个例外：

* **全速**（风扇模式 1）的时候服务**不看**每个风扇的数据，只记模式，所以这一项
  只核对模式。
* **显卡模式**不读回核对：它写进的是杂项设置表，机器要到重启才真的换过去，
  读回来的值说明不了什么，只看服务有没有收下。

真机验证（`cmd/agcheck verify`）：八项设置逐项改一次、读回核对、还原，全过。
灯光的写入在原厂档案 `LightingProfile.ini` 与注册表的 `boot_sound_switch`、
`FanControl\CurrentFanMode` 上都能看到——那是服务自己记下的状态，不是我们
自己记的。

## 架构

```
cmd/viper          命令行入口
cmd/agcheck        新客户端的真机验证：读、写、整机读、逐项改一次再还原
cmd/fanproto       风扇三个模式的取证工具
cmd/winkey         WIN_KEY 的语义与能不能写
cmd/gpucross       显卡遥测与原厂服务的读数对照
cmd/smbiosdump     机器身份信息的原始结构

internal/agentservice 原厂 AcerAgentService 的协议客户端
internal/machine      硬件层：原厂程序当主路，固件补它给不了的那几项
internal/hardware     ACPI 固件那一侧（补充，要管理员权限）
internal/wmi          WMI 客户端：实例解析、方法调用、参数类型自适应
internal/com          COM/OLE 互操作层（VARIANT、BSTR、SAFEARRAY、vtable）
internal/nvidia       显卡遥测：走显卡驱动的管理库
internal/quickaccess  原厂 Quick Access 服务客户端（蓝光灯、USB 充电）
internal/config       设置持久化
internal/app          用例层
```

规矩与 OpenPredator 项目一致：**硬件层不认识界面，界面层不直接调用 WMI，
中间夹着应用层。**

硬件层与协议层之间只有两个接口：`machine.Service`（对原厂服务）与
`machine.Extras`（对固件）。两边都不认识对方的实现，所以换掉任何一边都
不动另一边。

## 依赖哪些原厂进程

| 程序 | 端口 | 管什么 | 缺了会怎样 |
| :--- | :--- | :--- | :--- |
| `AcerAgentService` | 46933 | 灯光、灯效、风扇、性能档位、显卡模式、LCD 超频、Win 键锁、开机音效、背光超时、全部传感器读数 | 什么都做不了 |
| `AcerLightingService` | — | 真正控制键盘灯，`AcerAgentService` 把灯光设置转交给它 | 灯不会变色，其它设置仍可用 |
| `AcerServiceWrapper` | — | 看护进程，被杀掉的 `AcerAgentService` 由它拉起来 | 服务崩了不会自动恢复 |
| `AcerQAAgent` | 5141 | 蓝屏灯、USB 充电限值与开关、显示模式、暗色模式、屏保、任务栏自动隐藏 | 这些设置做不了，其余照常 |

显卡遥测是唯一不走原厂服务的例外：它走显卡驱动自带的 `nvml.dll`
（NVIDIA Management Library）。这个库是 **N 卡驱动的一部分**，不是别的程序，
Windows 本来就在提供它；没有 N 卡的机器读不到显卡遥测，其余功能照常工作。

## 命令行

```powershell
# 编译
go build -o bin/viper.exe ./cmd/viper

# 运行
bin\viper.exe status
```

输出示例：

```
instance: AcerAgentService
route  : 原厂服务 AcerAgentService

  Power supply      : on the power supply, charging

  CPU temperature   : 91 C
  CPU fan speed     : 2698 RPM
  GPU temperature   : 64 C
  GPU fan speed     : 2736 RPM
  Motherboard / VRM : 71 C
```

场景分区设置：原厂应用把「灯效 + 风扇」打包成**场景**，可以绑到各个性能
档位上，切档位时场景跟着走。

```powershell
viper scenario list                  # 列出场景与绑定
viper scenario save game             # 把当前的灯效与风扇存为场景
viper scenario apply game            # 应用场景
viper scenario bind performance game # 把场景绑到「性能」档
viper scenario unbind performance    # 解绑
viper scenario delete game           # 删除场景
```

## 管理员权限下的固件验证

固件那一侧的几项要管理员权限。`scripts\firmware-check.ps1` 会把它们逐项读
出来，并逐项改一次、读回核对、再改回原来的样子：

```powershell
# 以管理员身份打开 PowerShell
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\firmware-check.ps1
```

## 测试

```powershell
go test ./...
go test ./internal/machine/... ./internal/agentservice/... -cover
```

用例锁的是**实测得来的结论**，不是实现的复述：风扇的三个模式、灯光效果名
与颜色的往返、性能档位的五个编码、图形显示模式的编号、杂项设置的标识符、
颜色字与背光缓冲区的字节布局、SMBIOS 的解析、Quick Access 的加密——最后这
一条用的是原厂服务握手时返回的真实密文，改动加密实现会立刻让用例失败。

覆盖率：`internal/machine` 86.6%、`internal/agentservice` 93.4%、
`internal/app` 82.8%、`internal/hardware` 81.8%。

## 已知限制

* **锁 Windows 键在这台机器上做不到**：原厂服务收下请求却不生效，固件那一侧
  的档案类是个常量通道。Viper 如实报错。
* 蓝光灯的写入接口只接受 `0`–`4` 五个预设档位，本机当前是「自定义」
  （`5`），写入后无法用同一接口还原。
* 启动标志自定义需要固件支持，本机
  `HKLM\SOFTWARE\OEM\AcerAgentService\Customize_POST_Support` 为 `0`，
  原厂服务也回答 `SUPPORT_CUSTOM_BOOT_LOGO = 0`。
* 显卡的**超频能力**固件不报告，杂项设置表里只有 CPU 的那一项（`0x0A`）。
  原厂服务自己也不实现这一项（结果码 3）。Viper 因此报「显卡不支持超频」，
  不去拿「这台机器有独显」冒充「显卡能超频」。
* 官方应用自己保存的灯效档案在
  `C:\ProgramData\OEM\AcerLightingService\LightingProfile\LightingProfile.ini`，
  它与固件是两条独立的路径，Viper 不去改它。

## 参考

Viper 的功能范围与硬件语义参考了开源项目
[OpenPredator](https://github.com/KRWCLASSIC/OpenPredator)（MIT），
但实现完全独立，未复制其代码。

Acer、Nitro、Predator、PredatorSense、NitroSense 与 CoolBoost 是
Acer Inc. 的商标。Viper 与 Acer Inc. 无任何关联或背书关系。
