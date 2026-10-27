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

显卡遥测是唯一的例外：它走显卡驱动自带的 `nvml.dll`
（NVIDIA Management Library）。这个库是 **N 卡驱动的一部分**，不是
别的程序，Windows 本来就在提供它；没有 N 卡的机器读不到显卡遥测，
其余功能照常工作。驱动接口本身（`nvapi64.dll`）只导出一个
`nvapi_QueryInterface`，其余全靠数字编号调用，所以 Viper 选管理库
而不是驱动接口。

## provider 的完整能力面

`cmd/coverage` 把这个 provider 能给的东西全部扫了一遍，结论是
**硬件层在 WMI 这一侧已经没有未探明的空白**。

| 接口 | 覆盖情况 |
| :--- | :--- |
| `GetGamingSysInfo` | 一共只有 **5 个分类**有值（`0x01` CPU 温度、`0x02` CPU 风扇、`0x03` 主板温度、`0x06` 显卡风扇、`0x0A` 显卡温度），其余 12 个分类读回来都是 0。Viper 全读了 |
| 风扇 | 行为、转速、风扇表三个方法全部覆盖，读写都有 |
| 灯效 | 颜色、行为、亮度、背光缓冲区全部覆盖；三个读回方法（`GetGamingLED` / `GetGamingLEDColor` / `GetGamingLEDBehavior`）在固件里**读不到颜色**：`GetGamingLED` 返回的字节全是零，`GetGamingLEDColor` 与 `GetGamingLEDBehavior` 恒答 2 |
| 杂项设置 | `0x02` 图形显示模式、`0x06` 开机音效可读写；`0x01` 与 `0x08` 写入后没有任何可见效果 |
| 性能档案 | `GetGamingProfile` 与 `GetGamingProfileSetting` 能读，但语义还没解开（返回的是 1 和 2 这样的标志） |
| 动作类热键 | 只有 **132**（键盘背光）和 **7**（CoolBoost）有反应。Windows 菜单键在 HID 层，不在 WMI 面上 |
| 超频 | `Set/GetCPUOverclockingProfile` 未用，结构体格式还没逆出来。固件的杂项表里只有 CPU 的能力位（`0x0A`），显卡与显存的超频能力**固件根本不报告** |
| 显卡 / 显存超频 | 原厂应用有 **6 档显卡核心 + 6 档显存**的超频，但它走的不是 WMI，而是 `AcerAgentService` 的 `SOFTWARE\OEM\AcerAgentService\Overclock` 配置项。Viper 不走这一层，因此**完全不支持** |
| Turbo Key | 原厂硬件服务用 `SetGamingLEDBehaviorFunction` 做 Turbo Key 开关（LED 与按键一起开关），Viper 没有封装 |

剩下的缺口都不在 WMI 上，而是**别的层**：显卡遥测（使用率、频率）
要 NVAPI，Windows 菜单键要 HID 特性报告，启动标志要写 EFI 分区，
显卡与显存的超频要走原厂代理服务的配置项。

## 场景分区设置

原厂应用把「灯效 + 风扇」打包成**场景**，可以绑到各个性能档位上，
切档位时场景跟着走。Viper 实现了同一套机制：

```powershell
viper scenario list                  # 列出场景与绑定
viper scenario save game             # 把当前的灯效与风扇存为场景
viper scenario apply game            # 应用场景
viper scenario bind performance game # 把场景绑到「性能」档
viper scenario unbind performance    # 解绑
viper scenario delete game           # 删除场景
```

绑定之后，`viper power performance` 会连同场景一起应用。

存场景时，风扇模式与占空比、键盘亮度都**从机器上现读**，灯效与颜色
固件读不回来，取 Viper 自己的记录。固件在背光自动熄灭时会把亮度报成
零，所以零不被当成用户选的亮度。

### 灯光是怎么控制的

本机实测（`cmd/rgblayout`、`cmd/ledprobe`，需要管理员权限）：

- 固件里 `GetGamingLED` 读回来的字节**始终是全零**，`GetGamingLEDColor` 与
  `GetGamingLEDBehavior` 恒答 **2**，八种灯效的结果一模一样。**分区颜色读不回来。**
- 真正让键盘变色的是**原厂服务**（`AcerAgentService`，监听 46933）。让它设一次
  四个分区的纯红，键盘**立刻变红**。同一次写入用 WMI 读固件依然是全零，
  所以「读不到」不代表「没写进去」。
- 原厂服务**不走 WMI 写灯光**，它经命名管道
  `AcerLightingService_NamedPipe` 把设置交给 `AcerLightingService`：
  `SetKeyboardLightingMode`（全域）、`SetKeyboardLightingPerLEDLighting`
  （逐 LED），颜色是四个字节（`color: #%02X%02X%02X%02X`）。

因此 Viper 的灯光**调用原厂服务下发**，不逆向命名管道，也不依赖 WMI 的
`SetGamingLEDColor` —— 那条路原厂根本不用，写进去的东西没有验证手段。

其余设置原厂服务也都能写（`OPERATING_MODE`、`GPU_MODE`、`BACKLIGHT_30S`、
`LCD_OVERDRIVE`、`BOOT_SOUND`、`WIN_KEY` 实测均返回 `result: "0"`），
只有 `STICKY_KEY` 原厂自己也不支持。

## 显卡遥测

```
viper telemetry
  Graphics card     : NVIDIA GeForce RTX 4060 Laptop GPU
  Graphics usage    : 43 %
  Graphics clock    : 315 MHz (max 3105 MHz)
  Memory clock      : 6001 MHz
  Video memory      : 3.8 GB of 8.0 GB
```

读数与原厂服务（它走驱动接口 `nvapi64.dll`）**逐项吻合**：
使用率 35% 对 34%、图形频率 210 MHz 对 210、最高频率 3105 MHz 对 3105。
`cmd/gpucross` 是做这个对照的工具。

两个驱动细节：显存要走 `nvmlDeviceGetMemoryInfo`（第一版，三字段），
`_v2` 版本本机驱动直接返回"不支持"；性能状态 `nvmlDeviceGetPState`
这个版本没有导出，所以空闲状态改用使用率判断。

## 顶替原厂服务

官方 PredatorSense 应用**按端口号**连接本机的一个服务来读写机器状态。
Viper 可以提供同一个服务，官方应用就会改连 Viper。

```powershell
# 先停掉原厂服务，再让 Viper 顶上它的端口
viper-service.exe
```

服务默认监听 `127.0.0.1:46933`（原厂服务的端口），用 `-address` 可以
换到别的端口。协议是逆向出来的：

```
请求： "ACER" + uint32(小端) 包标识 + JSON 负载
应答： 纯 JSON 文档
包标识：0 初始化 / 10 监视数据 / 20 读设置 / 30 设备 / 40 灯效能力 / 100 写设置
```

应答的**字节格式**与原厂服务完全一致（jsoncpp 的 StyledWriter：Tab
缩进、成员按字节序排序、浮点用 `%.17g` 并补 `.0`），`cmd/padiff` 会把
两个服务的应答逐项对照。

对照结果（`cmd/padiff`，28 个请求）：

| 结果 | 数量 |
| :--- | :--- |
| 结构完全一致 | 25 |
| 字节级完全一致 | 2 |
| 有可解释差异 | 见下 |

有差异的三项：

| 差异 | 原因 |
| :--- | :--- |
| `CPU_MAX_FREQUENCY` 报 2419 而不是 5000 | 原厂服务报的是**睿频上限**，Windows 的标准接口只给标称频率（注册表 `~MHz` 与 `Win32_Processor.MaxClockSpeed` 都是 2419），Viper 不编造一个测不出来的数字 |
| `CPU_FREQUENCY` 报 2419 而不是实时频率 | 同上，Windows 没有给实时频率的标准接口 |
| `CPU_CORE_FREQUENCY` 是空数组 | 每核频率要靠性能计数器，Windows 的性能数据接口在本机初始化失败，Viper 不编造数字 |
| `WIFI_/ETHERNET_` 上下行速率是 0 | Viper 没有测量网络流量 |
| `gpu_max_frequency` 报 3105 而不是 5000 | 原厂服务把这个字段填成了**处理器的**频率，字段却叫显卡的。Viper 填显卡驱动自己报的最高频率，**这一项是 Viper 更准，不是原厂更准** |

灯效那一项两边本来就不同：原厂服务报的是**官方应用**最后写下的状态，
Viper 报的是 **Viper 自己的**配置。Viper 顶替之后，官方应用显示的
就会是 Viper 写的灯效，两边自动同步。

写入路径也验证过（会真实改硬件，随后还原）：写灯光 → 机器真的变成
`METEOR / 速度 7 / 方向 1 / 亮度 33 / 四色分区`；写开机音效 → 生效。

## 硬件层的新路线：原厂服务当主路

硬件层现在多了一条路：**主路是原厂服务**（`AcerAgentService`，监听
`127.0.0.1:46933`），固件那一侧只补它不管的几项。

```
原厂 AcerAgentService ──internal/agentservice（协议客户端）──▶ internal/machine
                                                                          │
                                            internal/hardware.Firmware ──┘
                                            （风扇加速、超频能力、背光缓冲区、复位灯效）
```

原厂服务以系统的身份运行，**程序不需要管理员权限就能用它**。本机实测：当前
会话没有管理员权限，固件那一侧连实例都找不到（`no AcerGamingFunction
instance found`），原厂服务这一侧却什么都能读。

原厂服务读得到的东西比固件多：

| 能力 | 固件（WMI） | 原厂服务 |
| :--- | :--- | :--- |
| 分区颜色 | ❌ 读回来永远是全零 | ✅ 四个分区的真实颜色 |
| 灯效 | ❌ 八个灯效的结果一模一样 | ✅ 效果名、速度、方向、亮度 |
| 显卡风扇组的行为 | ❌ 只报告主风扇组 | ✅ 两组都报 |
| 每核实时频率 | ❌ 性能计数器在本机初始化失败 | ✅ 32 核每一个都有 |
| 显卡使用率、网络上下行 | ❌ | ✅ |
| 睿频上限 | 2419（标称频率） | 5000（真实的睿频上限） |

风扇的三个模式是实测出来的，写在 `internal/machine` 的注释里：`mode` 为 `0`
是自动（2722 RPM），`1` 是全速（**7092 RPM**），`2` 是固定占空比（写成 50 时
落在 4400 RPM）。

这一层与协议层之间只有两个接口：`machine.Service` 与 `hardware.Firmware`，
两边都不认识对方的实现。细节见 `internal/machine/README.md`。

**这一层还没有接到命令行上**，命令行走的是旧的固件路线。接线是下一步的事。

## 服务自己记住的三项设置

有三个设置在固件里**根本没有接口**，问不回来：

| 设置 | 固件接口 | Viper 的做法 |
| :--- | :--- | :--- |
| `BATTERY_BOOST` | 没有 | 服务记住自己写下的值 |
| `SOUND_MODE` | 没有 | 服务记住自己写下的值 |
| `CUSTOM_BOOT_LOGO` | 没有（本机固件不支持自定义启动标志） | 服务记住自己写下的值 |

Viper **不拿别的东西冒充读回值**：这三项报的就是服务自己的记录，
从固件读回来的话只会是编造的。记录只存在服务进程的内存里，服务
重启后回到默认值。

`SUPPORT_OVERCLOCKING_MODE_CAPABILITY` 这一项**原厂服务自己也不实现**，
它的应答里没有数据，只有失败的代码。Viper 跟着原厂回答
`{"request":…,"result":"3"}`，不虚报一个「支持」。

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

同属 Quick Access 服务的还有显示模式、深色模式、透明效果、任务栏自动
隐藏与屏幕保护程序（`viper displaymode` / `darkmode` / `transparent` /
`taskbar` / `screensaver`），它们同样需要那个服务在跑。

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
| 性能档位：静音 / 均衡 / 性能 / 加速 / 节能 | ✅ |
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
| 电源与电池状态（供电、电量、充电中） | ✅ 走 Windows 内核，不需要管理员权限 |
| 显卡遥测（型号、使用率、频率、显存） | ✅ 走显卡驱动的管理库 |
| 场景分区设置（每个性能档位绑定自己的灯效与风扇） | ✅ |
| 超频能力探测 | ✅ 只有固件报告的 CPU 超频能力，固件没有显卡超频能力这一项 |
| CPU / GPU 超频配置 | 🚧 读取已通，写入待确认 |
| 配置持久化与开机还原 | 🚧 |
| 显示模式 / 深色模式 / 透明效果 / 任务栏自动隐藏 / 屏幕保护程序 | ✅ 走原厂 Quick Access 服务 |
| 单元测试（505 个用例，锁定全部实测结论） | ✅ |
| 图形界面 | 🚧 |
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
cmd/viper-service  原厂服务的替代实现
internal/com       COM/OLE 互操作层（VARIANT、BSTR、SAFEARRAY、vtable）
internal/wmi       WMI 客户端：实例解析、方法调用、参数类型自适应
internal/hardware  Acer 硬件后端：传感器、风扇、电源、灯效
internal/agentservice 原厂 AcerAgentService 的协议客户端
internal/machine   硬件层的新路线：原厂服务当主路，固件补它不管的几项
internal/nvidia    显卡遥测：走显卡驱动的管理库
internal/quickaccess 原厂 Quick Access 服务客户端（蓝光灯、USB 充电）
internal/service   原厂服务的协议实现（读写设置、监视数据）
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
| 性能档位编码 | **不是** MOF 暗示的 `0/1/2/3`。实测可用值只有 `0` 静音、`1` 均衡、`4` 性能、`5` 加速、`6` 节能；写 `2`/`3` 会被静默钳制回均衡。原厂服务返回的 `SUPPORT_OPERATING_MODE_CAPABILITY` 同样是 `[0,1,4,5,6]`，官方应用也把节能档暴露给用户 |
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
| `0x0B` | 性能档位 | `0` 静音、`1` 均衡、`4` 性能、`5` 加速、`6` 节能 |
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

四百零五个用例锁定的是**实测得来的结论**，不是实现的复述：性能档位的
五个编码、风扇行为的三个编码、八种灯效的编号、图形显示模式的编号、
杂项设置的标识符、颜色字与背光缓冲区的字节布局、SMBIOS 的解析、
jsoncpp 应答的字节格式、以及 Quick Access 的加密——最后这一条用的是
原厂服务握手时返回的真实密文，改动加密实现会立刻让用例失败。

## 已知限制

* 蓝光灯的写入接口只接受 `0`–`4` 五个预设档位，本机当前是
  「自定义」（`5`），写入后无法用同一接口还原。
* 启动标志自定义需要固件支持，本机
  `HKLM\SOFTWARE\OEM\AcerAgentService\Customize_POST_Support` 为 `0`，
  原厂服务也回答 `SUPPORT_CUSTOM_BOOT_LOGO = 0`。
* GPU 风扇组的行为读不回来，Viper 如实呈现而不伪造。
* 显卡的**超频能力**固件不报告，杂项设置表里只有 CPU 的那一项
  （`0x0A`）。Viper 因此报「显卡不支持超频」，不去拿「这台机器有独显」
  冒充「显卡能超频」。
* 电池加速、声音模式、启动标志这三项在固件里没有接口，Viper 的服务
  自己记住写下的值；这份记录只活在服务进程的内存里，重启后回到默认值。
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
