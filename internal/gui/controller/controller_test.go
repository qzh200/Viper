package controller_test

import (
	"errors"
	"testing"
	"time"

	"viper/internal/domain"
	"viper/internal/gui/controller"
	"viper/internal/gui/guitest"
	"viper/internal/gui/model"
	"viper/internal/gui/theme"
)

func TestTheFirstReadFillsTheSnapshot(t *testing.T) {
	machine := &guitest.Hardware{
		MonitorData:  domain.MonitorData{CPUTemperatureC: 55},
		Power:        domain.PowerTurbo,
		InstanceName: "AcerAgentService",
		RouteName:    "原厂服务 AcerAgentService",
	}
	ctrl := controller.New(machine, &guitest.Store{})

	overview := ctrl.Snapshot()
	if !overview.Connected {
		t.Fatal("the machine is not connected")
	}
	if overview.Instance != "AcerAgentService" {
		t.Errorf("the instance is %q, want AcerAgentService", overview.Instance)
	}
	if overview.Route != "原厂服务 AcerAgentService" {
		t.Errorf("the route is %q, want the service route", overview.Route)
	}
	if overview.PowerMode != domain.PowerTurbo {
		t.Errorf("the power mode is %q, want the turbo mode", overview.PowerMode)
	}
	if len(overview.Metrics) != 6 {
		t.Errorf("there are %d metrics, want 6", len(overview.Metrics))
	}
	if overview.Updated.IsZero() {
		t.Error("the snapshot has no time on it")
	}
}

func TestAMachineThatDoesNotAnswerIsReported(t *testing.T) {
	machine := &guitest.Hardware{Err: guitest.ErrHardware}
	ctrl := controller.New(machine, &guitest.Store{})

	overview := ctrl.Snapshot()
	if overview.Connected {
		t.Error("the machine is connected although it did not answer")
	}
	if overview.Error == "" {
		t.Error("the failure was not reported")
	}
	if len(overview.Metrics) != 0 {
		t.Errorf("there are %d metrics, want none", len(overview.Metrics))
	}
}

func TestAFailedWriteIsReportedAndKept(t *testing.T) {
	machine := &guitest.Hardware{SetErr: errors.New("机器没有答应")}
	ctrl := controller.New(machine, &guitest.Store{})

	var failure error
	if !ctrl.SetPowerMode(domain.PowerTurbo, func(err error) { failure = err }) {
		t.Fatal("the write was not queued")
	}
	ctrl.Work()

	if failure == nil {
		t.Fatal("the failure was not handed to the caller")
	}
	if got := ctrl.Snapshot().Error; got == "" {
		t.Error("the snapshot does not carry the failure")
	}
}

func TestAQueuedWriteIsAppliedByTheWorker(t *testing.T) {
	machine := &guitest.Hardware{}
	ctrl := controller.New(machine, &guitest.Store{})

	if !ctrl.SetPowerMode(domain.PowerPerformance, nil) {
		t.Fatal("the write was not queued")
	}
	// Nothing happens until the worker runs.
	if len(machine.Writes) != 0 {
		t.Fatalf("the machine was written to %d times before the worker ran", len(machine.Writes))
	}

	ctrl.Work()

	if len(machine.Writes) != 1 || machine.Writes[0] != domain.PowerPerformance {
		t.Errorf("the machine was written to %v, want the performance mode once", machine.Writes)
	}
	if got := ctrl.Snapshot().PowerMode; got != domain.PowerPerformance {
		t.Errorf("the snapshot holds %q, want the performance mode", got)
	}
}

func TestAWriteIsRefusedWhenTheQueueIsFull(t *testing.T) {
	ctrl := controller.New(&guitest.Hardware{}, &guitest.Store{})

	accepted := 0
	for i := 0; i < 64; i++ {
		if ctrl.SetPowerMode(domain.PowerTurbo, nil) {
			accepted++
		}
	}
	if accepted == 0 {
		t.Fatal("no write was accepted at all")
	}
	if accepted == 64 {
		t.Error("the queue took everything, it should have a limit")
	}
}

func TestTheWorkerClearsTheErrorAfterASuccessfulWrite(t *testing.T) {
	machine := &guitest.Hardware{SetErr: errors.New("机器没有答应")}
	ctrl := controller.New(machine, &guitest.Store{})

	ctrl.SetPowerMode(domain.PowerTurbo, nil)
	ctrl.Work()
	if ctrl.Snapshot().Error == "" {
		t.Fatal("the failure was not reported")
	}

	machine.SetErr = nil
	ctrl.SetPowerMode(domain.PowerBalanced, nil)
	ctrl.Work()
	if got := ctrl.Snapshot().Error; got != "" {
		t.Errorf("the snapshot still carries %q", got)
	}
}

func TestRefreshRereadsTheMachine(t *testing.T) {
	machine := &guitest.Hardware{MonitorData: domain.MonitorData{CPUTemperatureC: 40}}
	ctrl := controller.New(machine, &guitest.Store{})

	machine.MonitorData.CPUTemperatureC = 80
	ctrl.Refresh()

	metrics := ctrl.Snapshot().Metrics
	if len(metrics) == 0 || metrics[0].Value != "80 °C" {
		t.Errorf("the metrics are %v, want the new temperature", metrics)
	}
}

func TestTheBackgroundWorkerKeepsTheSnapshotFresh(t *testing.T) {
	machine := &guitest.Hardware{}
	ctrl := controller.New(machine, &guitest.Store{})
	defer ctrl.Stop()

	changed := make(chan struct{}, 1)
	ctrl.Start(time.Millisecond, func() {
		select {
		case changed <- struct{}{}:
		default:
		}
	})

	select {
	case <-changed:
	case <-time.After(5 * time.Second):
		t.Fatal("the background worker never reported a change")
	}
}

func TestStoppingTwiceIsHarmless(t *testing.T) {
	ctrl := controller.New(&guitest.Hardware{}, &guitest.Store{})
	ctrl.Stop()
	ctrl.Stop()
}

func TestTheAppearanceModeComesFromTheStore(t *testing.T) {
	store := &guitest.Store{Mode: "dark"}
	ctrl := controller.New(&guitest.Hardware{}, store)

	if got := ctrl.ThemeMode(); got != theme.ModeDark {
		t.Errorf("the mode is %q, want the dark one", got)
	}
}

func TestAnAppearanceModeNobodyKnowsFallsBackToTheAutomaticOne(t *testing.T) {
	ctrl := controller.New(&guitest.Hardware{}, &guitest.Store{Mode: "sepia"})
	if got := ctrl.ThemeMode(); got != theme.ModeAuto {
		t.Errorf("the mode is %q, want the automatic one", got)
	}
}

func TestAnAppearanceModeIsSaved(t *testing.T) {
	store := &guitest.Store{}
	ctrl := controller.New(&guitest.Hardware{}, store)

	if err := ctrl.SetThemeMode(theme.ModeLight); err != nil {
		t.Fatal(err)
	}
	if store.Mode != "light" {
		t.Errorf("the store holds %q, want light", store.Mode)
	}
	if len(store.Saved) != 1 {
		t.Errorf("the store was written %d times, want once", len(store.Saved))
	}
}

func TestAnUnknownAppearanceModeIsRefused(t *testing.T) {
	ctrl := controller.New(&guitest.Hardware{}, &guitest.Store{})
	if err := ctrl.SetThemeMode(theme.Mode("sepia")); err == nil {
		t.Error("an unknown mode was accepted")
	}
}

func TestAFailureToSaveIsHandedBack(t *testing.T) {
	store := &guitest.Store{Err: errors.New("disk is full")}
	ctrl := controller.New(&guitest.Hardware{}, store)

	if err := ctrl.SetThemeMode(theme.ModeDark); err == nil {
		t.Error("the failure to save was swallowed")
	}
}

func TestAControllerWithoutAStoreStillWorks(t *testing.T) {
	ctrl := controller.New(&guitest.Hardware{}, nil)

	if got := ctrl.ThemeMode(); got != theme.ModeAuto {
		t.Errorf("the mode is %q, want the automatic one", got)
	}
	if err := ctrl.SetThemeMode(theme.ModeDark); err != nil {
		t.Errorf("saving without a store failed: %v", err)
	}
}

func TestAMachineThatOnlyRefusesThePowerMode(t *testing.T) {
	machine := &powerOnlyFailure{}
	ctrl := controller.New(machine, &guitest.Store{})

	overview := ctrl.Snapshot()
	if !overview.Connected {
		t.Fatal("the machine is not connected although it answered")
	}
	if overview.Error == "" {
		t.Error("the refusal of the power mode was not reported")
	}
	if len(overview.Metrics) != 6 {
		t.Errorf("there are %d metrics, want the readings anyway", len(overview.Metrics))
	}
}

// powerOnlyFailure 是一台只在读性能档位时失败的机器。
type powerOnlyFailure struct {
	guitest.Hardware
}

func (m *powerOnlyFailure) PowerMode() (domain.PowerMode, error) {
	return domain.PowerBalanced, errors.New("档位读不出来")
}

func TestTheSamplesAreKept(t *testing.T) {
	machine := &guitest.Hardware{MonitorData: domain.MonitorData{CPUTemperatureC: 50}}
	ctrl := controller.New(machine, &guitest.Store{})

	if got := ctrl.Samples(); got != 1 {
		t.Errorf("there are %d samples, want the one of the first read", got)
	}

	for i := 0; i < 5; i++ {
		ctrl.Refresh()
	}
	if got := ctrl.Samples(); got != 6 {
		t.Errorf("there are %d samples, want 6", got)
	}
}

func TestTheSamplesDoNotGrowForever(t *testing.T) {
	machine := &guitest.Hardware{MonitorData: domain.MonitorData{CPUTemperatureC: 50}}
	ctrl := controller.New(machine, &guitest.Store{})

	for i := 0; i < model.HistoryLimit*2; i++ {
		ctrl.Refresh()
	}
	if got := ctrl.Samples(); got != model.HistoryLimit {
		t.Errorf("there are %d samples, want at most %d", got, model.HistoryLimit)
	}
}

func TestTheSnapshotIsACopy(t *testing.T) {
	ctrl := controller.New(&guitest.Hardware{}, &guitest.Store{})

	overview := ctrl.Snapshot()
	overview.Error = "scratched"

	if ctrl.Snapshot().Error == "scratched" {
		t.Error("the snapshot handed out the live one")
	}
}

func TestTheModelOfTheSnapshotIsTheSharedOne(t *testing.T) {
	ctrl := controller.New(&guitest.Hardware{}, &guitest.Store{})
	var overview model.Overview = ctrl.Snapshot()
	if len(overview.PowerModes) != len(domain.AvailablePowerModes()) {
		t.Errorf("the snapshot offers %d power modes, want %d",
			len(overview.PowerModes), len(domain.AvailablePowerModes()))
	}
}

func TestSmoothingInterpolatesWhenEnabled(t *testing.T) {
	machine := &guitest.Hardware{MonitorData: domain.MonitorData{CPUTemperatureC: 40}}
	ctrl := controller.New(machine, &guitest.Store{})

	ctrl.SetSmoothing(true)
	machine.MonitorData.CPUTemperatureC = 80
	ctrl.Refresh()

	// 开启平滑后，Refresh 设置目标为 80，但尚未推进平滑步进，初始仍为 40
	if got := ctrl.Snapshot().Monitor.CPUTemperatureC; got != 40 {
		t.Errorf("before step, got CPU temp %d, want 40", got)
	}

	// 步进一步平滑
	if !ctrl.StepSmooth(0.2) {
		t.Fatal("expected StepSmooth to report changes")
	}

	// 40 + (80-40)*0.2 = 48
	if got := ctrl.Snapshot().Monitor.CPUTemperatureC; got != 48 {
		t.Errorf("after 1 step, got CPU temp %d, want 48", got)
	}

	// 持续推进直到收敛
	for ctrl.StepSmooth(0.2) {
	}

	if got := ctrl.Snapshot().Monitor.CPUTemperatureC; got != 80 {
		t.Errorf("after converging, got CPU temp %d, want 80", got)
	}
}

func TestLightingControlAndDebounce(t *testing.T) {
	machine := &guitest.Hardware{
		LightingConfig: domain.RGBConfig{
			Effect:     domain.EffectStatic,
			Brightness: 50,
		},
		BacklightTimeoutState: false,
	}
	ctrl := controller.New(machine, &guitest.Store{})
	defer ctrl.Stop()

	// 1. 测试 SetLighting
	targetCfg := domain.RGBConfig{
		Effect:     domain.EffectWave,
		Brightness: 90,
	}
	if !ctrl.SetLighting(targetCfg, nil) {
		t.Fatal("SetLighting enqueue failed")
	}
	ctrl.Work()
	if len(machine.LightWrites) != 1 || machine.LightWrites[0].Effect != domain.EffectWave {
		t.Fatalf("expected 1 light write with EffectWave, got %+v", machine.LightWrites)
	}

	// 2. 测试 SetLightingDebounced
	ctrl.SetLightingDebounced(domain.RGBConfig{Brightness: 30}, 20*time.Millisecond, nil)
	// 在防抖时间内再次触发，覆盖之前的
	ctrl.SetLightingDebounced(domain.RGBConfig{Brightness: 85}, 20*time.Millisecond, nil)
	time.Sleep(50 * time.Millisecond)
	ctrl.Work()
	last := machine.LightWrites[len(machine.LightWrites)-1]
	if last.Brightness != 85 {
		t.Errorf("expected debounced brightness 85, got %d", last.Brightness)
	}

	// 3. 测试 ResetLighting
	if !ctrl.ResetLighting(nil) {
		t.Fatal("ResetLighting enqueue failed")
	}
	ctrl.Work()
	if machine.ResetLightCalls != 1 {
		t.Errorf("expected ResetLightCalls 1, got %d", machine.ResetLightCalls)
	}

	// 4. 测试 SetBacklightTimeout
	if !ctrl.SetBacklightTimeout(true, nil) {
		t.Fatal("SetBacklightTimeout enqueue failed")
	}
	ctrl.Work()
	if len(machine.TimeoutWrites) != 1 || !machine.TimeoutWrites[0] {
		t.Errorf("expected timeout true write, got %v", machine.TimeoutWrites)
	}
}

func TestHighFrequencyLightingWritesCoalesce(t *testing.T) {
	machine := &guitest.Hardware{}
	ctrl := controller.New(machine, &guitest.Store{})
	defer ctrl.Stop()

	// 模拟调色板极高频拖拽（例如 100 次连续写入）
	for i := 1; i <= 100; i++ {
		ok := ctrl.SetLighting(domain.RGBConfig{
			Brightness: uint64(i),
		}, nil)
		if !ok {
			t.Fatalf("write %d was unexpectedly refused", i)
		}
	}

	ctrl.Work()

	// 100 次高频写入应当被合并为极少量实际底层硬件写入（只有最初在途和最终待决）
	if len(machine.LightWrites) == 0 {
		t.Fatal("expected at least one write to hardware")
	}
	if len(machine.LightWrites) > 2 {
		t.Errorf("expected at most 2 hardware writes due to coalescing, got %d", len(machine.LightWrites))
	}
	// 无论合并了几次，最后一次写入硬件的一定是最终目标值（100）
	lastWrite := machine.LightWrites[len(machine.LightWrites)-1]
	if lastWrite.Brightness != 100 {
		t.Errorf("expected final brightness 100, got %d", lastWrite.Brightness)
	}
}

func TestFanControlAndCoalescing(t *testing.T) {
	machine := &guitest.Hardware{}
	ctrl := controller.New(machine, &guitest.Store{})
	defer ctrl.Stop()

	// 1. 测试 SetFanConfig 高频合并
	for i := 1; i <= 50; i++ {
		ok := ctrl.SetFanConfig(domain.FanConfig{
			CPUMode:    domain.FanCustom,
			CPUPercent: i,
		}, nil)
		if !ok {
			t.Fatalf("fan write %d was unexpectedly refused", i)
		}
	}
	ctrl.Work()
	if len(machine.FanWrites) == 0 || len(machine.FanWrites) > 2 {
		t.Errorf("expected 1 or 2 fan writes, got %d", len(machine.FanWrites))
	}
	lastFan := machine.FanWrites[len(machine.FanWrites)-1]
	if lastFan.CPUPercent != 50 {
		t.Errorf("expected final CPUPercent 50, got %d", lastFan.CPUPercent)
	}

	// 2. 测试 SetFanConfigDebounced
	ctrl.SetFanConfigDebounced(domain.FanConfig{CPUMode: domain.FanAuto}, 20*time.Millisecond, nil)
	ctrl.SetFanConfigDebounced(domain.FanConfig{CPUMode: domain.FanMax}, 20*time.Millisecond, nil)
	time.Sleep(50 * time.Millisecond)
	ctrl.Work()
	lastDebouncedFan := machine.FanWrites[len(machine.FanWrites)-1]
	if lastDebouncedFan.CPUMode != domain.FanMax {
		t.Errorf("expected debounced fan mode FanMax, got %v", lastDebouncedFan.CPUMode)
	}

	// 3. 测试 SetCoolBoost
	if !ctrl.SetCoolBoost(true, nil) {
		t.Fatal("SetCoolBoost failed to enqueue")
	}
	ctrl.Work()
	if len(machine.CoolBoostWrites) != 1 || !machine.CoolBoostWrites[0] {
		t.Errorf("expected coolboost true, got %v", machine.CoolBoostWrites)
	}
}

func TestFailureCallbacks(t *testing.T) {
	expectedErr := errors.New("hardware error")
	machine := &guitest.Hardware{SetErr: expectedErr}
	ctrl := controller.New(machine, &guitest.Store{})
	defer ctrl.Stop()

	var gotErr error
	ctrl.SetLighting(domain.RGBConfig{Brightness: 50}, func(err error) {
		gotErr = err
	})
	ctrl.Work()
	if gotErr == nil || gotErr.Error() != expectedErr.Error() {
		t.Errorf("expected %v, got %v", expectedErr, gotErr)
	}

	var gotFanErr error
	ctrl.SetFanConfig(domain.FanConfig{CPUMode: domain.FanAuto}, func(err error) {
		gotFanErr = err
	})
	ctrl.Work()
	if gotFanErr == nil || gotFanErr.Error() != expectedErr.Error() {
		t.Errorf("expected %v, got %v", expectedErr, gotFanErr)
	}
}

func TestPerformanceSwitchesAndGPUMode(t *testing.T) {
	machine := &guitest.Hardware{
		GPUModeState:      domain.GPUModeHybrid,
		DiscreteGPUState:  false,
		LCDOverdriveState: false,
		WinKeyLockedState: false,
	}
	ctrl := controller.New(machine, &guitest.Store{})
	defer ctrl.Stop()

	// 1. 测试显卡模式切换
	if !ctrl.SetGPUMode(domain.GPUModeDiscrete, nil) {
		t.Fatal("SetGPUMode failed to enqueue")
	}
	ctrl.Work()
	if len(machine.GPUModeWrites) != 1 || machine.GPUModeWrites[0] != domain.GPUModeDiscrete {
		t.Errorf("expected GPUModeDiscrete write, got %v", machine.GPUModeWrites)
	}

	// 2. 测试 LCD 超频开关
	if !ctrl.SetLCDOverdrive(true, nil) {
		t.Fatal("SetLCDOverdrive failed to enqueue")
	}
	ctrl.Work()
	if len(machine.LCDOverdriveWrites) != 1 || !machine.LCDOverdriveWrites[0] {
		t.Errorf("expected LCDOverdrive true, got %v", machine.LCDOverdriveWrites)
	}

	// 3. 测试 Win 键锁定开关
	if !ctrl.SetWinKeyLocked(true, nil) {
		t.Fatal("SetWinKeyLocked failed to enqueue")
	}
	ctrl.Work()
	if len(machine.WinKeyLockedWrites) != 1 || !machine.WinKeyLockedWrites[0] {
		t.Errorf("expected WinKeyLocked true, got %v", machine.WinKeyLockedWrites)
	}

	// 快照核验
	snap := ctrl.Snapshot()
	if snap.GPUMode != domain.GPUModeDiscrete {
		t.Errorf("expected snapshot GPUMode discrete, got %v", snap.GPUMode)
	}
	if !snap.LCDOverdrive {
		t.Error("expected snapshot LCDOverdrive true")
	}
	if !snap.WinKeyLocked {
		t.Error("expected snapshot WinKeyLocked true")
	}
}
