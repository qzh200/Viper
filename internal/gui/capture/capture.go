package capture

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Window interface {
	CapturePage() ([]byte, error)
}

const DefaultAttempts = 200

const DefaultInterval = 50 * time.Millisecond

func Save(win Window, path string) error {
	png, err := win.CapturePage()
	if err != nil {
		return fmt.Errorf("capture: %w", err)
	}
	return write(path, png)
}

func WaitAndSave(win Window, path string) error {
	return WaitAndSaveWithin(win, path, DefaultAttempts, DefaultInterval)
}

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
