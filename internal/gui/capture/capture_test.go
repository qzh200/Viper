package capture_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"viper/internal/gui/capture"
)

// fakeWindow 是一扇会照着剧本说话的窗口。
type fakeWindow struct {
	png []byte
	err error
	// calls 记下截了多少次。
	calls int
	// failFirst 是开头失败的次数。
	failFirst int
}

func (w *fakeWindow) CapturePage() ([]byte, error) {
	w.calls++
	if w.calls <= w.failFirst {
		return nil, errors.New("画面还没有尺寸")
	}
	if w.err != nil {
		return nil, w.err
	}
	return w.png, nil
}

func TestTheWindowIsWrittenToAFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "viper.png")
	win := &fakeWindow{png: []byte("not really a png")}

	if err := capture.Save(win, path); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "not really a png" {
		t.Errorf("the file holds %q, want the bytes of the window", data)
	}
}

func TestTheDirectoryOfThePictureIsMade(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pictures", "viper.png")
	win := &fakeWindow{png: []byte("bytes")}

	if err := capture.Save(win, path); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}

func TestAFailureToCaptureIsHandedBack(t *testing.T) {
	path := filepath.Join(t.TempDir(), "viper.png")
	win := &fakeWindow{err: errors.New("窗口已经关了")}

	if err := capture.Save(win, path); err == nil {
		t.Error("the failure was swallowed")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("a picture was left behind although nothing was captured")
	}
}

func TestTheWindowIsWaitedFor(t *testing.T) {
	path := filepath.Join(t.TempDir(), "viper.png")
	win := &fakeWindow{png: []byte("bytes"), failFirst: 3}

	if err := capture.WaitAndSave(win, path); err != nil {
		t.Fatal(err)
	}
	if win.calls != 4 {
		t.Errorf("the window was captured %d times, want 4", win.calls)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}

func TestAWindowThatNeverAppearsIsGivenUpOn(t *testing.T) {
	path := filepath.Join(t.TempDir(), "viper.png")
	win := &fakeWindow{failFirst: 100}

	err := capture.WaitAndSaveWithin(win, path, 3, 0)
	if err == nil {
		t.Error("the wait gave up too easily")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("a picture was left behind although nothing was captured")
	}
}

func TestAPlaceThatCannotHoldAPictureIsReported(t *testing.T) {
	dir := t.TempDir()
	// 一个普通文件不能当目录用。
	blocker := filepath.Join(dir, "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	win := &fakeWindow{png: []byte("bytes")}
	if err := capture.Save(win, filepath.Join(blocker, "viper.png")); err == nil {
		t.Error("the picture was written although the place cannot hold it")
	}
}

func TestAPlaceThatIsAlreadyADirectoryIsReported(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "viper.png")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatal(err)
	}

	win := &fakeWindow{png: []byte("bytes")}
	if err := capture.Save(win, target); err == nil {
		t.Error("the picture was written over a directory")
	}
}
