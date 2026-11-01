package widgets

import (
	"math"
	"sync"
	"time"

	"github.com/egoist/mygo/ui"
)

const (
	fanBlades   = 7
	fanMaxRPM   = 6000
	fanMaxTurns = 0.75
)

type fanTracker struct {
	mu        sync.Mutex
	lastTimes map[ui.Color]time.Time
	angles    map[ui.Color]float64
}

var globalFanTracker = &fanTracker{
	lastTimes: make(map[ui.Color]time.Time),
	angles:    make(map[ui.Color]float64),
}

func (ft *fanTracker) nextAngle(tint ui.Color, rpm int, now time.Time) float64 {
	ft.mu.Lock()
	defer ft.mu.Unlock()

	last := ft.lastTimes[tint]
	ft.lastTimes[tint] = now
	if last.IsZero() {
		return 0
	}

	dt := now.Sub(last).Seconds()
	if dt <= 0 || dt > 0.1 {
		dt = 0.016
	}

	turns := fanTurns(rpm)
	cur := ft.angles[tint]
	cur += turns * 2 * math.Pi * dt
	cur = math.Mod(cur, 2*math.Pi)
	ft.angles[tint] = cur
	return cur
}

const gaugeSweep = 270

const gaugeStart = 135

type RadialGauge struct {
	Value     float64
	Low, High float64
	ValueText string
	Label     string
	Tint      ui.Color
	Size      float32
}

func Gauge(c *ui.Context, spec RadialGauge) ui.Element {
	t := c.Theme()
	size := spec.Size
	if size <= 0 {
		size = 132
	}

	dial := ui.Box(c).Size(size, size).Draw(func(p *ui.Painter, r ui.Rect) {
		cx, cy := r.X+r.W/2, r.Y+r.H/2
		radius := min(r.W, r.H)/2 - 7
		thickness := float32(8)

		p.StrokePath(gaugeArc(cx, cy, radius, gaugeStart, gaugeSweep), thickness, t.Border)

		fraction := 0.0
		if spec.High > spec.Low {
			fraction = (spec.Value - spec.Low) / (spec.High - spec.Low)
		}
		fraction = clamp01(fraction)
		if fraction > 0 {
			p.StrokePath(gaugeArc(cx, cy, radius, gaugeStart, gaugeSweep*float32(fraction)), thickness, spec.Tint)
		}
	})

	dial.Children(func() {
		ui.Column(c).Fill().Center().Children(func() {
			ui.Text(c, spec.ValueText).FontSize(26).Bold().TextColor(spec.Tint).NoWrap()
			if spec.Label != "" {
				ui.Text(c, spec.Label).FontSize(11).TextColor(t.TextMuted).NoWrap()
			}
		})
	})
	return dial
}

func gaugeArc(cx, cy, radius, start, sweep float32) *ui.Path {
	path := &ui.Path{}
	path.Cap(ui.RoundCap).Join(ui.RoundJoin)
	const steps = 48
	for i := 0; i <= steps; i++ {
		angle := (start + sweep*float32(i)/steps) * math.Pi / 180
		px := cx + radius*float32(math.Cos(float64(angle)))
		py := cy + radius*float32(math.Sin(float64(angle)))
		if i == 0 {
			path.MoveTo(px, py)
			continue
		}
		path.LineTo(px, py)
	}
	return path
}

func Fan(c *ui.Context, rpm int, size float32, tint ui.Color) ui.Element {
	return ui.Box(c).Size(size, size).Draw(func(p *ui.Painter, r ui.Rect) {
		t := c.Theme()
		cx, cy := r.X+r.W/2, r.Y+r.H/2
		radius := min(r.W, r.H) / 2

		p.StrokePath(circle(cx, cy, radius*0.94), 1.6, tint.Alpha(0.30))

		if rpm <= 0 {
			p.FillPath(circle(cx, cy, radius*0.13), tint.Alpha(0.45))
			return
		}

		p.AnimationFrame()

		turns := fanTurns(rpm)
		spin := globalFanTracker.nextAngle(tint, rpm, p.Now())

		alpha := float32(0.85)
		if turns > fanMaxTurns*0.75 {
			alpha = 0.65
		}

		for blade := 0; blade < fanBlades; blade++ {
			theta := spin + float64(blade)*2*math.Pi/fanBlades
			p.FillPath(fanBlade(cx, cy, radius*0.84, float32(theta)), tint.Alpha(alpha))
		}
		p.FillPath(circle(cx, cy, radius*0.24), t.Surface)
		p.FillPath(circle(cx, cy, radius*0.13), tint)
	}).Label("风扇")
}

func fanTurns(rpm int) float64 {
	if rpm <= 0 {
		return 0
	}
	speed := float64(rpm) / fanMaxRPM * fanMaxTurns
	if speed > fanMaxTurns {
		speed = fanMaxTurns
	}
	return speed
}

func fanBlade(cx, cy, radius, angle float32) *ui.Path {
	cos, sin := float32(math.Cos(float64(angle))), float32(math.Sin(float64(angle)))
	at := func(x, y float32) (float32, float32) {
		return cx + x*cos - y*sin, cy + x*sin + y*cos
	}

	r := radius
	path := &ui.Path{}

	hx, hy := at(0, -r*0.26)
	path.MoveTo(hx, hy)

	c1x, c1y := at(r*0.26, -r*0.48)
	c2x, c2y := at(r*0.44, -r*0.74)
	tx, ty := at(r*0.30, -r*0.95)
	path.CubeTo(c1x, c1y, c2x, c2y, tx, ty)

	c1x, c1y = at(r*0.23, -r*1.04)
	c2x, c2y = at(r*0.00, -r*1.00)
	tx, ty = at(-r*0.09, -r*0.88)
	path.CubeTo(c1x, c1y, c2x, c2y, tx, ty)

	c1x, c1y = at(-r*0.13, -r*0.64)
	c2x, c2y = at(-r*0.06, -r*0.42)
	path.CubeTo(c1x, c1y, c2x, c2y, hx, hy)
	path.Close()
	return path
}

func clamp01(value float64) float64 {
	switch {
	case value < 0:
		return 0
	case value > 1:
		return 1
	default:
		return value
	}
}
