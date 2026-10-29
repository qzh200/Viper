// Package app 协调配置存储、硬件后端及 QuickAccess 服务，提供顶层业务接口。
package app

import (
	"fmt"
	"sync"

	"viper/internal/config"
	"viper/internal/domain"
	"viper/internal/machine"
	"viper/internal/quickaccess"
)

type abilities interface {
	domain.ReadingsReader
	domain.PowerModeReader
	domain.GraphicsModeReader
	domain.LightingReader
	domain.LightingConfig
	domain.LightingWriter
	domain.LightingReset
	domain.FansReader
	domain.FansWriter
	domain.CoolBoostReader
	domain.SwitchesReader

	Monitor() (domain.MonitorData, error)
}

const (
	routeService = "原厂服务 AcerAgentService"
)

// App 是 Viper 控制中心的核心应用实例，负责统一组织硬件交互与本地配置管理。
type App struct {
	mu sync.Mutex

	backend abilities

	openBackend func() (domain.Gaming, string, error)

	serviceAddress string

	route string

	closed bool

	store *config.Store

	quickClient QuickAccess

	quickOptions quickaccess.Options
}

var serviceAddress string

// SetServiceAddress 设置原厂服务 AcerAgentService 的默认监听地址。
func SetServiceAddress(address string) { serviceAddress = address }

// New 创建使用默认用户配置的 App 实例。
func New() (*App, error) {
	store, err := config.Open()
	if err != nil {
		return nil, err
	}
	return NewWithStore(store), nil
}

// NewWithStore 使用指定的配置存储实例构建 App。
func NewWithStore(store *config.Store) *App {
	return &App{store: store, serviceAddress: serviceAddress}
}

// NewWithBackend 使用预置的硬件控制后端与配置存储构建 App。
func NewWithBackend(store *config.Store, backend abilities) *App {
	return &App{store: store, backend: backend}
}

// NewWithQuickAccess 使用预置的 QuickAccess 客户端构建 App。
func NewWithQuickAccess(store *config.Store, client QuickAccess) *App {
	return &App{store: store, quickClient: client}
}

// SetQuickAccessOptions 设置 QuickAccess 连接选项。
func (a *App) SetQuickAccessOptions(options quickaccess.Options) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.quickOptions = options
}

// Close 关闭应用并断开后端硬件服务连接。
func (a *App) Close() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.backend != nil {
		a.backend.Close()
		a.backend = nil
	}
	a.closed = true
}

func (a *App) hardware() (abilities, error) {
	return a.connect(func() (abilities, error) {
		open := a.openBackend
		if open == nil {
			open = func() (domain.Gaming, string, error) {
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

func openMachine(address string) (domain.Gaming, string, error) {
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

// SetBackendOpener 自定义硬件后端连接工厂函数，主要用于单元测试与模拟注入。
func (a *App) SetBackendOpener(open func() (domain.Gaming, string, error)) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.openBackend = open
}

// Route 返回当前与硬件后端建立连接的通道名称。
func (a *App) Route() string {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.route == "" {
		return "还没连接"
	}
	return a.route
}

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

// SettingsPath 返回当前生效配置文件的物理路径。
func (a *App) SettingsPath() string { return a.store.Path() }
