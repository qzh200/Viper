package widgets

import (
	"math"

	"github.com/egoist/mygo/ui"
)

// 风扇的几何与转速参数。
const (
	// fanBlades 是风扇的叶片数。
	fanBlades = 7
	// fanMaxRPM 是画到最快时的转速：真正的风扇能到六七千转，
	// 但画太快就成了一团糊，因此按这个上限折算成看得清的速度。
	fanMaxRPM = 6000
	// fanMaxTurns 是折算出来的最高转速，每秒几圈。
	fanMaxTurns = 3.0
)

// gaugeSweep 是仪表弧线的角度：底下留一道口子。
const gaugeSweep = 270

// gaugeStart 是仪表弧线的起点，屏幕坐标里从左下往右下绕。
const gaugeStart = 135

// RadialGauge 画一个圆表：外圈是一道弧线，中间是读数。
//
// 弧线按 Value 在 Low 与 High 之间的位置画，Tint 是弧线与数字的颜色。
type RadialGauge struct {
	// Value 是读数。
	Value float64
	// Low 与 High 是读数的上下限。
	Low, High float64
	// ValueText 是中间那行大字。
	ValueText string
	// Label 是中间那行小字。
	Label string
	// Tint 是弧线与数字的颜色。
	Tint ui.Color
	// Size 是表盘的边长。
	Size float32
}

// Gauge 画一个圆表。
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

// gaugeArc 画一段圆弧：从 start 度开始，顺时针转过 sweep 度。
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

// Fan 画一台转起来的风扇，转多快由转速决定。
//
// 转速为零时风扇停着；转起来的每一帧都由 MyGo 自己再要一帧。
func Fan(c *ui.Context, rpm int, size float32, tint ui.Color) ui.Element {
	return ui.Box(c).Size(size, size).Draw(func(p *ui.Painter, r ui.Rect) {
		t := c.Theme()
		cx, cy := r.X+r.W/2, r.Y+r.H/2
		radius := min(r.W, r.H) / 2

		// 外圈。
		p.StrokePath(circle(cx, cy, radius*0.94), 1.6, tint.Alpha(0.30))

		if rpm <= 0 {
			// 停着的时候只画轴。
			p.FillPath(circle(cx, cy, radius*0.13), tint.Alpha(0.45))
			return
		}

		p.AnimationFrame()

		// 转多快由转速说了算：转速越高转得越快。
		// 角度要先在 float64 里绕回一圈再交给 float32：直接用时钟的毫秒数去算，
		// 单精度存不下那么大的数，七片叶子会挤到同一个角度上去。
		turns := fanTurns(rpm)
		spin := math.Mod(float64(p.Now().UnixMilli())/1000*turns*2*math.Pi, 2*math.Pi)

		// 转得快的时候淡一点，再补一圈残影，看着才像在使劲转。
		alpha := float32(0.85)
		if turns > fanMaxTurns*0.6 {
			alpha = 0.55
			p.StrokePath(circle(cx, cy, radius*0.60), 3, tint.Alpha(0.18))
		}

		// 七片叶子分七次填：mygo 的路径只填第一段，分成一条路径会少六片。
		for blade := 0; blade < fanBlades; blade++ {
			theta := spin + float64(blade)*2*math.Pi/fanBlades
			p.FillPath(fanBlade(cx, cy, radius*0.84, float32(theta)), tint.Alpha(alpha))
		}
		// 轴心：先拿底色挖一圈，再点上轴心，叶片与轴心之间就有了空隙。
		p.FillPath(circle(cx, cy, radius*0.24), t.Surface)
		p.FillPath(circle(cx, cy, radius*0.13), tint)
	}).Label("风扇")
}

// fanTurns 把转速折算成画出来的圈数：转速越高转得越快。
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

// fanBlade 画一片叶子：一片朝外甩的桨叶。
//
// 桨叶根部窄、在外端变宽，叶尖是圆的，整体顺着转动方向偏一点，
// 这样七片排在一起才像一台风扇，而不是一朵花。
func fanBlade(cx, cy, radius, angle float32) *ui.Path {
	cos, sin := float32(math.Cos(float64(angle))), float32(math.Sin(float64(angle)))
	at := func(x, y float32) (float32, float32) {
		return cx + x*cos - y*sin, cy + x*sin + y*cos
	}

	r := radius
	path := &ui.Path{}

	// 根部，靠着轴心。
	hx, hy := at(0, -r*0.26)
	path.MoveTo(hx, hy)

	// 前缘：顺着转动的方向甩出去。
	c1x, c1y := at(r*0.26, -r*0.48)
	c2x, c2y := at(r*0.44, -r*0.74)
	tx, ty := at(r*0.30, -r*0.95)
	path.CubeTo(c1x, c1y, c2x, c2y, tx, ty)

	// 叶尖：圆滑地收回来。
	c1x, c1y = at(r*0.23, -r*1.04)
	c2x, c2y = at(r*0.00, -r*1.00)
	tx, ty = at(-r*0.09, -r*0.88)
	path.CubeTo(c1x, c1y, c2x, c2y, tx, ty)

	// 背面：收回到轴心。
	c1x, c1y = at(-r*0.13, -r*0.64)
	c2x, c2y = at(-r*0.06, -r*0.42)
	path.CubeTo(c1x, c1y, c2x, c2y, hx, hy)
	path.Close()
	return path
}

// clamp01 把比例限制在 0 到 1 之间。
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
