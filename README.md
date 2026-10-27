# Viper

Acer 游戏本硬件控制中心的 Go 重写版。

它的硬件层走**原厂服务**：官方 PredatorSense 的后台服务
（`AcerAgentService`）是这台机器上真正管着灯光、风扇、性能档位与传感器
读数的那一路，Viper 作为它的客户端去问，**不需要管理员权限**。原厂服务不
管的那几项（风扇加速、风扇表、超频能力、背光原始缓冲区）才走 ACPI 固件
那一侧，那一侧需要管理员权限。

```
                 ┌────────────────────────────────────────────┐
                 │  Viper                                     │
                 │                                            │
   原厂服务 ───▶ │  internal/agentservice ─▶ internal/machine │ ──▶ 硬件层接口
 (46933, SYSTEM) │   （协议客户端）            （新硬件层）    │      ↑
                 │                                            │      │
   固件 ───────▶ │  internal/hardware（兜底，需要管理员权限）──┘      │
 (root\wmi)      │                                            │      │
                 │  internal/quickaccess（蓝光灯、USB 充电等）───┘      │
                 └────────────────────────────────────────────┘
```

## 现在的状态

| 部分 | 状态 |
| :--- | :--- |
| 新硬件层 `internal/machine` + `internal/agentservice` | ✅ 已完成，真机验证通过 |
| 命令行接到新硬件层 | 🚧 还没接，命令行现在走旧的固件那一侧 |
| 顶替模式 `viper-service` | ✅ 保留，用固件那一侧当数据源 |
| 图形界面 | 🚧 以后再说 |

## 原厂服务比固件多读得到什么

固件那一侧读不到的东西，原厂服务都读得到。下表是本机实测的：

| 能力 | 固件（WMI） | 原厂服务 |
| :--- | :--- | :--- |
| 分区颜色 | ❌ 读回来永远是全零 | ✅ 四个分区的真实颜色 |
| 灯效 | ❌ 八个灯效的结果一模一样 | ✅ 效果名、速度、方向、亮度 |
| 显卡风扇组的行为 | ❌ 只报告主风扇组 | ✅ 两组都报 |
| 每核实时频率 | ❌ 性能计数器在本机初始化失败 | ✅ 32 核每一个都有 |
| 显卡使用率 | ❌ | ✅ |
| 网络上下行 | ❌ | ✅ |
| 内存占用与频率 | ❌ | ✅ |
| 睿频上限 | 2419（标称频率） | 5000（真实的睿频上限） |

实测这一段的时候，本机会话**没有管理员权限**：固件那一侧连实例都找不到
（`no AcerGamingFunction instance found`），原厂服务这一侧却什么都能读。
这就是换路线的最大收益——它不以管理员身份运行。

## 实测得到的编码

这一节里的每一条都是在这台机器上量出来的，不是照着文档推的。

**风扇**（`FAN_CONTROL`）：顶层 `mode` 是开关，每个风扇的
`fan_custom_auto` 与 `fan_custom_speed` 是它的行为与占空比。

| 写法 | 实测 |
| :--- | :--- |
| `mode: 0` | 自动，控制器跑自己的曲线（2722 RPM） |
| `mode: 1` | 全速，**7092 RPM** |
| `mode: 2` + `fan_custom_auto: 0` + `fan_custom_speed: 50` | 固定占空比，**4400 RPM** |

**性能档位**（`OPERATING_MODE`）：编号是 `0` 静音、`1` 均衡、`4` 性能、
`5` 加速、`6` 节能，与固件那一侧的编号一样。

**显卡模式**（`GPU_MODE`）：服务从零开始数，`0` 混合、`1` 仅独显、
`2` 自动。切换图形模式要重启才生效，与官方应用的行为一致。

**开关类**（`WIN_KEY`、`LCD_OVERDRIVE`、`BOOT_SOUND`、`BACKLIGHT_30S`）：
形状都是 `{"status": 0 或 1}`。`WIN_KEY` 的编号 1 表示**这个键还能用**，
所以锁住的时候是 0。

**灯光**（`LIGHTING`）：四个分区各一个颜色（`#RRGGBB`），加上效果名、速度、
方向与亮度。写亮度时要把四个分区一起写回去，否则原厂服务会把颜色清掉。

**服务不实现的几项**：`SUPPORT_OVERCLOCKING_MODE_CAPABILITY` 的结果是 3，
`STICKY_KEY` 的写入结果是 4。这两项原厂自己也不支持，粘滞键走
`SystemParametersInfo`，超频能力走固件那一侧。

## 协议

原厂服务的协议是逆向出来的，两边都要用它。

```
请求： "ACER" + uint32(小端) 包标识 + JSON 负载
应答： 纯 JSON 文档 {"data":…,"request":"…","result":"0"}
包标识：0 初始化 / 10 监视数据 / 20 读设置 / 30 设备 / 40 灯效能力 / 100 写设置
结果码：0 成功 / 3 服务不实现 / 4 服务没有执行
```

应答的**字节格式**与原厂服务完全一致（jsoncpp 的 StyledWriter：Tab 缩进、
成员按字节序排序、浮点用 `%.17g` 并补 `.0`），`cmd/padiff` 会把两个服务的
应答逐项对照。

对照结果（`cmd/padiff`，28 个请求）：

| 结果 | 数量 |
| :--- | :--- |
| 结构完全一致 | 25 |
| 字节级完全一致 | 2 |
| 有可解释差异 | 见下 |

| 差异 | 原因 |
| :--- | :--- |
| `CPU_MAX_FREQUENCY` 报 2419 而不是 5000 | 原厂服务报的是**睿频上限**，Windows 的标准接口只给标称频率（注册表 `~MHz` 与 `Win32_Processor.MaxClockSpeed` 都是 2419），Viper 不编造一个测不出来的数字 |
| `CPU_FREQUENCY` 报 2419 而不是实时频率 | 同上，Windows 没有给实时频率的标准接口 |
| `CPU_CORE_FREQUENCY` 是空数组 | 每核频率要靠性能计数器，Windows 的性能数据接口在本机初始化失败，Viper 不编造数字 |
| `WIFI_/ETHERNET_` 上下行速率是 0 | Viper 没有测量网络流量 |
| `gpu_max_frequency` 报 3105 而不是 5000 | 原厂服务把这个字段填成了**处理器的**频率，字段却叫显卡的。Viper 填显卡驱动自己报的最高频率，**这一项是 Viper 更准，不是原厂更准** |

## 架构

```
cmd/viper          命令行入口
cmd/viper-service  顶替模式：提供原厂服务的同一个端口
cmd/agcheck        新客户端的真机验证：读、写、整机读
cmd/fanproto       风扇三个模式的取证工具
cmd/pacapture      录原厂服务的应答
cmd/padiff         原厂服务与 Viper 服务的逐项对照
cmd/gpucross       显卡遥测与原厂服务的读数对照
cmd/smbiosdump     机器身份信息的原始结构

internal/agentservice 原厂 AcerAgentService 的协议客户端
internal/machine      硬件层的新路线：原厂服务当主路，固件补它不管的几项
internal/hardware     ACPI 固件那一侧（兜底 + 顶替模式的数据源）
internal/wmi          WMI 客户端：实例解析、方法调用、参数类型自适应
internal/com          COM/OLE 互操作层（VARIANT、BSTR、SAFEARRAY、vtable）
internal/nvidia       显卡遥测：走显卡驱动的管理库
internal/quickaccess  原厂 Quick Access 服务客户端（蓝光灯、USB 充电）
internal/service      顶替模式的协议实现（读写设置、监视数据）
internal/config       设置持久化
internal/app          用例层
```

规矩与 OpenPredator 项目一致：**硬件层不认识界面，界面层不直接调用 WMI，
中间夹着应用层。**

新硬件层与协议层之间只有两个接口：`machine.Service`（对原厂服务）与
`hardware.Firmware`（对固件）。两边都不认识对方的实现，所以换掉任何一边都
不动另一边。

## 依赖哪些原厂进程

| 程序 | 端口 | 管什么 | 缺了会怎样 |
| :--- | :--- | :--- | :--- |
| `AcerAgentService` | 46933 | 灯光、灯效、风扇、性能档位、显卡模式、LCD 超频、Win 键锁、开机音效、背光超时、全部传感器读数 | 新硬件层什么都做不了 |
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
instance: AcerGamingFunction.InstanceName="ACPI\\PNP0C14\\APGe_0"

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

## 顶替模式

官方 PredatorSense 应用**按端口号**连接本机的一个服务来读写机器状态。
Viper 可以提供同一个服务，官方应用就会改连 Viper。

```powershell
# 先停掉原厂服务，再让 Viper 顶上它的端口
viper-service.exe
```

服务默认监听 `127.0.0.1:46933`（原厂服务的端口），用 `-address` 可以换到
别的端口。这个模式的数据源是固件那一侧（顶替模式的前提就是原厂服务已经
停掉，客户端那条路这时不可用），所以它需要管理员权限。

服务自己记住的设置：有三项设置**固件里根本没有接口**，服务读不到，也没法
从机器上问出来。

| 设置 | 服务的做法 |
| :--- | :--- |
| `BATTERY_BOOST` | 记住自己写过的值，没写过就报原厂的值 `1` |
| `SOUND_MODE` | 记住自己写过的值，没写过就报 `{"function":"DTS","mode":0}` |
| `CUSTOM_BOOT_LOGO` | 记住自己写过的值，没写过就报 `{"imageSource":""}` |

**这三个值是服务自己的记录，不是从固件读回来的值。** 它们只活在服务进程的
内存里，服务一重启就回到原厂的默认值。

## Quick Access 服务

原厂 Quick Access 服务（`AcerQAAgent.exe`，服务名 `AcerQAAgentSvis`）监听
`127.0.0.1:5141`，管着蓝光灯、USB 关机充电、充电阈值、显示模式、深色模式、
透明效果、任务栏自动隐藏与屏幕保护程序。

```
握手： 先做 WebSocket 升级，再发 AuthPacket（PacketType = 1），
      负载是 AES-128-ECB 加密的 {"Question":…, "Key":…}
请求： {"PacketType":2,"Version":1,"Session":…,"Command":…,"Action":"Get"|"Set","Param1":…}
应答： {"PacketType":3,"Command":…,"Action":…,"Result":{"Value":…,"Text":…}}
```

两把密钥与挑战串在原厂应用里是硬编码的，`internal/quickaccess` 按同样的
方式实现。实测把该服务停掉后，蓝光灯与 USB 充电立刻报「服务不可达」，
服务重新启动后全部恢复。

## 测试

```powershell
go test ./...
go test ./internal/machine/... ./internal/agentservice/... -cover
```

用例锁的是**实测得来的结论**，不是实现的复述：风扇的三个模式、灯光效果名
与颜色的往返、性能档位的五个编码、图形显示模式的编号、杂项设置的标识符、
颜色字与背光缓冲区的字节布局、SMBIOS 的解析、jsoncpp 应答的字节格式、
Quick Access 的加密——最后这一条用的是原厂服务握手时返回的真实密文，改动
加密实现会立刻让用例失败。全仓六百多个测试函数。

新硬件层的覆盖率：`internal/machine` 90.3%，`internal/agentservice` 93.4%。

## 已知限制

* **命令行还没有接到新硬件层上**，它现在走旧的固件那一侧，所以命令行上的
  灯光颜色仍然是从 Viper 自己的配置里读出来的。
* 蓝光灯的写入接口只接受 `0`–`4` 五个预设档位，本机当前是「自定义」
  （`5`），写入后无法用同一接口还原。
* 启动标志自定义需要固件支持，本机
  `HKLM\SOFTWARE\OEM\AcerAgentService\Customize_POST_Support` 为 `0`，
  原厂服务也回答 `SUPPORT_CUSTOM_BOOT_LOGO = 0`。
* 显卡的**超频能力**固件不报告，杂项设置表里只有 CPU 的那一项（`0x0A`）。
  原厂服务自己也不实现这一项（结果码 3）。Viper 因此报「显卡不支持超频」，
  不去拿「这台机器有独显」冒充「显卡能超频」。
* 电池加速、声音模式、启动标志这三项在固件里没有接口，顶替模式的服务自己
  记住写下的值；这份记录只活在服务进程的内存里，重启后回到默认值。
* 风扇加速、风扇表、超频能力、背光原始缓冲区这几项原厂服务给不了，需要
  固件那一侧，也就是需要管理员权限。够不到时报的是「这一项需要固件接口，
  原厂服务不提供，需要管理员权限」，不拿别的东西冒充。
* 官方应用自己保存的灯效档案在
  `C:\ProgramData\OEM\AcerLightingService\LightingProfile\LightingProfile.ini`，
  它与固件是两条独立的路径，Viper 不去改它。

## 参考

Viper 的功能范围与硬件语义参考了开源项目
[OpenPredator](https://github.com/KRWCLASSIC/OpenPredator)（MIT），
但实现完全独立，未复制其代码。

Acer、Nitro、Predator、PredatorSense、NitroSense 与 CoolBoost 是
Acer Inc. 的商标。Viper 与 Acer Inc. 无任何关联或背书关系。
