// Package capture 把 Viper 的窗口画成 PNG。
//
// 截图走的是 MyGo 自己的通道：原生界面由 MyGo 画，它把画布交回来，
// 因此拿到的就是界面本身，不必借操作系统的桌面截屏。
package capture

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Window 是能被截下来的窗口。*mygo.Window 满足这个接口。
type Window interface {
	// CapturePage 返回窗口当前画面的 PNG 字节。
	CapturePage() ([]byte, error)
}

// DefaultAttempts 是等待画面就绪时尝试的次数。
//
// 一次等五十毫秒，两百次就是十秒：窗口刚开出来的时候要一会儿才有尺寸。
const DefaultAttempts = 200

// DefaultInterval 是两次尝试之间隔多久。
const DefaultInterval = 50 * time.Millisecond

// Save 把窗口当前的样子写成 PNG 文件。
func Save(win Window, path string) error {
	png, err := win.CapturePage()
	if err != nil {
		return fmt.Errorf("capture: %w", err)
	}
	return write(path, png)
}

// WaitAndSave 等窗口画好第一帧，再把它写成 PNG 文件。
//
// 窗口刚创建时还没有尺寸，画不出东西来，所以这里重试几次。
func WaitAndSave(win Window, path string) error {
	return WaitAndSaveWithin(win, path, DefaultAttempts, DefaultInterval)
}

// WaitAndSaveWithin 是 WaitAndSave，只是重试的次数与间隔由调用方定。
func WaitAndSaveWithin(win Window, path string, attempts int, interval time.Duration) error {
	if attempts <= 0 {
		attempts = DefaultAttempts
	}
	var lastErr error
	for attempt := 0; attempt < attempts; attempt++ {
		png, err := win.CapturePage()
		if err == nil && len(png) > 0 {
			return write(path, png)
		}
		lastErr = err
		if interval > 0 {
			time.Sleep(interval)
		}
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("画面始终没有就绪")
	}
	return fmt.Errorf("capture: 等了 %d 次也没有画出来: %w", attempts, lastErr)
}

// write 把 PNG 字节写到文件里，先写临时文件再改名，免得留下半张图。
func write(path string, png []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("capture: 建 %s 失败: %w", dir, err)
	}

	temp, err := os.CreateTemp(dir, "viperui-*.png")
	if err != nil {
		return fmt.Errorf("capture: 在 %s 里建临时文件失败: %w", dir, err)
	}
	name := temp.Name()
	defer os.Remove(name)

	if _, err := temp.Write(png); err != nil {
		temp.Close()
		return fmt.Errorf("capture: 写 %s 失败: %w", name, err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("capture: 写 %s 失败: %w", name, err)
	}
	if err := os.Rename(name, path); err != nil {
		return fmt.Errorf("capture: 写 %s 失败: %w", path, err)
	}
	return nil
}
