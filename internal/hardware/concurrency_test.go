package hardware

import (
	"sync"
	"time"

	"viper/internal/wmi"
)

// 服务的每个连接各有一个 goroutine，它们共用同一个后端。写设置和写完
// 之后的读回校验必须是一整段，不然两个请求会交叉：四个分区的颜色会
// 来自两份不同的配置，键盘上真的就花了。

// recordingProvider 记录每次写分区时用的颜色。
//
// 第一个写分区的调用会在这里等一下，等第二个请求也发起写入才放行。
// 这样两个请求一定会交叉，除非后端把一整段操作锁住了。
type recordingProvider struct {
	mutex sync.Mutex
	// order 记录了每次写分区时用的颜色。
	order []uint64
	// arrived 记录了有多少个写分区的调用到了。
	arrived int
	// release 放行被挡住的调用。
	release chan struct{}
	// once 保证只放行一次。
	once sync.Once
}

func newRecordingProvider() *recordingProvider {
	return &recordingProvider{release: make(chan struct{})}
}

func (p *recordingProvider) Call(class, method, instancePath string, args map[string]wmi.Value) (map[string]wmi.Value, error) {
	if method == "SetGamingLEDColor" {
		value := args["gmInput"].UInt64()
		p.mutex.Lock()
		p.order = append(p.order, value&0xFFFFFF00)
		p.arrived++
		reached := p.arrived
		p.mutex.Unlock()

		if reached == 1 {
			// 第一个到的在这里等，等第二个请求也发起写入。
			// 后端把一整段操作锁住时，第二个请求进不来，这里会等到超时，
			// 所以等一会儿就放行。
			select {
			case <-p.release:
			case <-time.After(200 * time.Millisecond):
			}
		} else {
			p.once.Do(func() { close(p.release) })
		}
	}
	return map[string]wmi.Value{"gmOutput": wmi.Uint64(0)}, nil
}

func (p *recordingProvider) Instances(class string) ([]string, error) {
	return []string{class}, nil
}

func (p *recordingProvider) Close() {}

// colours 列出记录的分区颜色。
func (p *recordingProvider) colours() []uint64 {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	out := make([]uint64, len(p.order))
	copy(out, p.order)
	return out
}

// zoneColours 是两份配置各自的四个分区颜色。
//
// 两份配置给每个分区不同的颜色，所以看颜色就知道它来自哪一份配置。
// 颜色字的排布是蓝、绿、红从高到低，参考 colorWord。
func zoneColours() (first []uint64, second []uint64) {
	build := func(base RGBColor) []uint64 {
		out := make([]uint64, 0, LightingZones)
		for i := 0; i < LightingZones; i++ {
			colour := RGBColor{R: base.R, G: base.G, B: base.B + byte(i)}
			out = append(out, (uint64(colour.B)<<24)|(uint64(colour.G)<<16)|(uint64(colour.R)<<8))
		}
		return out
	}
	return build(RGBColor{R: 0xC7, G: 0x00, B: 0x87}), build(RGBColor{R: 0x00, G: 0xAE, B: 0xC7})
}

// matches 报告颜色是不是这一份配置里的。
func matches(colour uint64, colours []uint64) bool {
	for _, c := range colours {
		if c == colour {
			return true
		}
	}
	return false
}
