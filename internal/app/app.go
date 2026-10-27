// Package app is the use case layer of Viper.
//
// It joins the hardware backend with the stored settings. The settings that
// do not belong to the gaming hardware, such as the blue light shield, are
// read from the service of the original software.
//
// 硬件层只有一条路：原厂程序。灯光、灯效、风扇、性能档位与传感器读数都问
// 原厂服务，它以系统的身份运行，不用管理员权限。
package app

import (
	"fmt"
	"sync"

	"viper/internal/config"
	"viper/internal/hardware"
	"viper/internal/machine"
	"viper/internal/quickaccess"
)

// abilities is the set of abilities of the machine that the program uses.
//
// The program does not report the table of the fans, so it depends on the
// small interfaces instead of the whole of the machine.
type abilities interface {
	hardware.ReadingsReader
	hardware.PowerModeReader
	hardware.GraphicsModeReader
	hardware.LightingReader
	hardware.LightingConfig
	hardware.LightingWriter
	hardware.LightingReset
	hardware.FansReader
	hardware.FansWriter
	hardware.CoolBoostReader
	hardware.SwitchesReader
	// Monitor reads everything the machine reports about itself: the
	// temperatures, the loads, the clocks, the memory and the network.
	Monitor() (hardware.MonitorData, error)
}

// The routes the program can reach the machine by.
const (
	// routeService goes through the service of the original software. It is
	// the road of the command line, and it needs no rights of an
	// administrator.
	routeService = "原厂服务 AcerAgentService"
)

// App is the running program.
type App struct {
	// mu 看守着程序按需打开的连接。游戏硬件和 Quick Access 服务都是
	// 第一次用到相关设置时才打开，所以不需要管理员权限的设置不会去
	// 碰硬件。这把锁保证同时到来的两次调用不会各开一条连接，然后把
	// 其中一条漏掉。
	mu sync.Mutex
	// backend 是到游戏硬件的连接。
	backend abilities
	// openBackend 打开到机器的连接。它是可变的，所以用例可以放一个替身
	// 在真机器的位置上。
	openBackend func() (hardware.Gaming, string, error)
	// serviceAddress 是连原厂服务用的地址，空的就是本机默认的端口。
	serviceAddress string
	// route 记着程序走的是哪条路：原厂服务还是固件那一侧。
	route string
	// closed 记录程序是否已经关闭。关闭之后的程序不再打开硬件。
	closed bool
	// store 保存两次运行之间的设置。
	store *config.Store
	// quickClient 是与 Quick Access 服务的会话。它由测试注入，程序
	// 自己每次调用都开一个新的会话。
	quickClient QuickAccess
	// quickOptions change the behaviour of the session with the Quick
	// Access service.
	quickOptions quickaccess.Options
}

// serviceAddress 是程序连原厂服务用的地址。
//
// 命令行只有一个 --service 选项，它要在程序开始时就定下来，所以它放在这里。
var serviceAddress string

// SetServiceAddress changes the address of the service of the original
// software.
//
// 它要在程序开始时调用，早于任何一次连接。
func SetServiceAddress(address string) { serviceAddress = address }

// New opens the stored settings.
//
// The connection to the gaming hardware is opened on first use, the
// settings of the original software do not need the rights that the
// hardware needs.
func New() (*App, error) {
	store, err := config.Open()
	if err != nil {
		return nil, err
	}
	return NewWithStore(store), nil
}

// NewWithStore opens the program with the given settings.
//
// It is used by the tests, so that they do not touch the settings of the
// user.
func NewWithStore(store *config.Store) *App {
	return &App{store: store, serviceAddress: serviceAddress}
}

// NewWithBackend opens the program with the given hardware.
//
// It is used by the tests, so that they do not need the rights of an
// administrator to talk to the firmware.
func NewWithBackend(store *config.Store, backend abilities) *App {
	return &App{store: store, backend: backend}
}

// NewWithQuickAccess opens the program with the given session of the
// Quick Access service.
//
// It is used by the tests, so that they do not need the service of the
// original software.
func NewWithQuickAccess(store *config.Store, client QuickAccess) *App {
	return &App{store: store, quickClient: client}
}

// SetQuickAccessOptions changes the way the program opens a session with
// the Quick Access service.
//
// It has to be called before the first session is opened, an address
// without the service of the original software makes every call of the
// service fail.
//
// 这个入口让 quickOptions 不再是一个永远为零的字段。
func (a *App) SetQuickAccessOptions(options quickaccess.Options) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.quickOptions = options
}

// Close releases the hardware.
func (a *App) Close() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.backend != nil {
		a.backend.Close()
		a.backend = nil
	}
	a.closed = true
}

// hardware opens the connection to the gaming hardware.
//
// The connection is only made when a hardware setting is used, because
// opening it takes a moment and the settings of the original software do
// not need it.
func (a *App) hardware() (abilities, error) {
	return a.connect(func() (abilities, error) {
		open := a.openBackend
		if open == nil {
			open = func() (hardware.Gaming, string, error) {
				return openMachine(a.serviceAddress)
			}
		}

		backend, route, err := open()
		if err != nil {
			return nil, err
		}
		a.route = route
		return backend, nil
	})
}

// openMachine opens the connection to the machine.
//
// 硬件层只有一条路：原厂 AcerAgentService。它不在的时候**没有第二条路**可
// 走——直接操控固件那一整条路线已经删掉了。所以原厂服务不在，就把实情报给
// 用的人。
//
// 第二个结果是程序走的是哪条路，命令行会把它报出来。
func openMachine(address string) (hardware.Gaming, string, error) {
	backend, err := machine.New(machine.Options{
		Address: address,
	})
	if err != nil {
		return nil, "", fmt.Errorf("%w"+
			"\n  原厂程序不在，这一层没有第二条路可走。",
			err)
	}
	return backend, routeService, nil
}

// SetBackendOpener changes the way the program opens the connection to the
// machine.
//
// It has to be called before the first connection is opened. The second
// result is the name of the route the program is on, it is reported to the
// user so that they know which road they are on.
//
// 这个入口让用例可以把一台机器的替身放在真机器的位置上，不用去碰真的
// 机器。
func (a *App) SetBackendOpener(open func() (hardware.Gaming, string, error)) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.openBackend = open
}

// Route returns the way the program reaches the machine.
//
// The connection is opened on first use, so before that the answer is that
// the route is not known yet.
func (a *App) Route() string {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.route == "" {
		return "还没连接"
	}
	return a.route
}

// connect opens the connection to the gaming hardware once.
//
// 连接由锁守着，同时到来的两次调用不会开出两条连接然后漏掉其中一条。
func (a *App) connect(open func() (abilities, error)) (abilities, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.closed {
		return nil, fmt.Errorf("app: 程序已经关闭")
	}
	if a.backend != nil {
		return a.backend, nil
	}
	backend, err := open()
	if err != nil {
		return nil, fmt.Errorf("app: 打开游戏硬件失败: %w", err)
	}
	a.backend = backend
	return backend, nil
}

// SettingsPath returns the location of the settings file.
func (a *App) SettingsPath() string { return a.store.Path() }
