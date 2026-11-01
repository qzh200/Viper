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

func SetServiceAddress(address string) { serviceAddress = address }

func New() (*App, error) {
	store, err := config.Open()
	if err != nil {
		return nil, err
	}
	return NewWithStore(store), nil
}

func NewWithStore(store *config.Store) *App {
	return &App{store: store, serviceAddress: serviceAddress}
}

func NewWithBackend(store *config.Store, backend abilities) *App {
	return &App{store: store, backend: backend}
}

func NewWithQuickAccess(store *config.Store, client QuickAccess) *App {
	return &App{store: store, quickClient: client}
}

func (a *App) SetQuickAccessOptions(options quickaccess.Options) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.quickOptions = options
}

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

func (a *App) SetBackendOpener(open func() (domain.Gaming, string, error)) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.openBackend = open
}

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

func (a *App) SettingsPath() string { return a.store.Path() }
