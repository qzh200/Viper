package hardware

// The overclocking functions of the provider are deliberately not wrapped.
// This file carries the notes about them.
//
// CPU 的超频（SetCPUOverclockingProfile 和 GetCPUOverclockingProfile）没有
// 封装，原因有三个：
//
//  1. 这个方法的参数是一个结构体，结构体里每个字段的位置、宽度和单位都还
//     没有弄清楚。电压和倍频写错会直接伤到硬件，宁可不做。
//  2. 读回这个结构体需要 internal/wmi 支持 ReturnCode 和 OCStructure 这两个
//     输出名，这部分由 internal/wmi 的负责人在做。
//  3. 在结构体没有弄清楚之前，写进去的东西没有办法核对，写完也不知道固件
//     收下了没有。
//
// GetGamingProfileSetting 同样没有封装。这个方法可以安全地只读调用，但是
// 它的语义至今没有解开：对 0 到 0x20 的每一个查询都读回一张 1 或者 2 的稠密
// 表，而改性能档位对它没有任何影响，所以它不是性能档位的开关。
//
// SetGamingProfileSetting 也没有封装：这个字里还带着别的开关，写它有可能
// 顺手改掉用户没有要求改的东西。
