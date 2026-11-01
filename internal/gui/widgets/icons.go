package widgets

import (
	"math"

	"github.com/egoist/mygo/ui"
)

type Icon int

const (
	IconCPU Icon = iota
	IconGPU
	IconMemory
	IconThermometer
	IconFan
	IconBolt
	IconActivity
	IconGear
	IconClock
	IconDisplay
	IconSun
	IconMoon
)

func IconElement(c *ui.Context, icon Icon, size float32, color ui.Color) ui.Element {
	return ui.Box(c).Size(size, size).Draw(func(p *ui.Painter, r ui.Rect) {
		icon.draw(p, r, color)
	}).Label(icon.name())
}

func (i Icon) name() string {
	switch i {
	case IconCPU:
		return "处理器"
	case IconGPU:
		return "显卡"
	case IconMemory:
		return "内存"
	case IconThermometer:
		return "温度"
	case IconFan:
		return "风扇"
	case IconBolt:
		return "供电"
	case IconActivity:
		return "负载"
	case IconGear:
		return "设置"
	case IconClock:
		return "时钟"
	case IconDisplay:
		return "屏幕"
	case IconSun:
		return "亮色"
	case IconMoon:
		return "暗色"
	default:
		return "图标"
	}
}

func (i Icon) draw(p *ui.Painter, r ui.Rect, color ui.Color) {
	side := r.W
	if r.H < side {
		side = r.H
	}
	unit := side / 24
	x := func(v float32) float32 { return r.X + v*unit }
	y := func(v float32) float32 { return r.Y + v*unit }

	const stroke = 1.7

	switch i {
	case IconCPU:
		p.Stroke(ui.Rect{X: x(4), Y: y(4), W: 16 * unit, H: 16 * unit}, color, 2*unit, stroke)
		p.Stroke(ui.Rect{X: x(9), Y: y(9), W: 6 * unit, H: 6 * unit}, color, unit, stroke)
		for _, t := range []float32{9, 12, 15} {
			p.Line(x(t), y(1), x(t), y(4), stroke, color)
			p.Line(x(t), y(20), x(t), y(23), stroke, color)
			p.Line(x(1), y(t), x(4), y(t), stroke, color)
			p.Line(x(20), y(t), x(23), y(t), stroke, color)
		}

	case IconGPU:
		p.Stroke(ui.Rect{X: x(2), Y: y(4), W: 20 * unit, H: 16 * unit}, color, 2*unit, stroke)
		p.StrokePath(circle(x(8), y(12), 3.6*unit), stroke, color)
		for _, at := range []float32{9, 12, 15} {
			p.Line(x(15), y(at), x(19), y(at), stroke, color)
		}

	case IconMemory:
		p.Stroke(ui.Rect{X: x(2), Y: y(6), W: 20 * unit, H: 12 * unit}, color, 2*unit, stroke)
		for _, at := range []float32{7, 10.5, 14, 17.5} {
			p.Line(x(at), y(6), x(at), y(18), 1.2, color)
		}
		for _, at := range []float32{6, 12, 18} {
			p.Line(x(at), y(18), x(at), y(21), stroke, color)
		}

	case IconThermometer:
		p.Line(x(12), y(4.5), x(12), y(14), 1.8, color)
		p.FillPath(circle(x(12), y(16.5), 3.4*unit), color)
		for _, at := range []float32{8, 11} {
			p.Line(x(15), y(at), x(18), y(at), 1.3, color)
		}

	case IconFan:
		p.StrokePath(circle(x(12), y(12), 9*unit), 1.5, color)
		for blade := 0; blade < 3; blade++ {
			angle := float64(blade)*math.Pi*2/3 - math.Pi/2
			cos, sin := math.Cos(angle), math.Sin(angle)
			at := func(px, py float32) (float32, float32) {
				dx, dy := (px-12)*unit, (py-12)*unit
				return x(12) + dx*float32(cos) - dy*float32(sin),
					y(12) + dx*float32(sin) + dy*float32(cos)
			}
			path := &ui.Path{}
			bx, by := at(12, 10)
			path.MoveTo(bx, by)
			c1x, c1y := at(17, 10.5)
			c2x, c2y := at(18, 5.5)
			tx, ty := at(14, 4)
			path.CubeTo(c1x, c1y, c2x, c2y, tx, ty)
			c3x, c3y := at(13, 5)
			c4x, c4y := at(11.5, 7.5)
			path.CubeTo(c3x, c3y, c4x, c4y, bx, by)
			path.Close()
			p.FillPath(path, color)
		}
		p.FillPath(circle(x(12), y(12), 2*unit), color)

	case IconBolt:
		path := &ui.Path{}
		path.MoveTo(x(13.5), y(2))
		path.LineTo(x(5), y(13))
		path.LineTo(x(11), y(13))
		path.LineTo(x(10), y(22))
		path.LineTo(x(19), y(10.5))
		path.LineTo(x(13), y(10.5))
		path.Close()
		p.FillPath(path, color)

	case IconActivity:
		for i, height := range []float32{5, 9.5, 6.5, 12} {
			left := 3.5 + float32(i)*4.5
			p.Fill(ui.Rect{
				X: x(left),
				Y: y(19 - height),
				W: 2.8 * unit,
				H: height * unit,
			}, color, 1)
		}
		p.Line(x(3), y(19.5), x(21), y(19.5), 1.4, color)

	case IconGear:
		p.StrokePath(circle(x(12), y(12), 6.2*unit), stroke, color)
		p.StrokePath(circle(x(12), y(12), 2.4*unit), 1.4, color)
		for tooth := 0; tooth < 8; tooth++ {
			angle := float64(tooth) * math.Pi / 4
			cos, sin := math.Cos(angle), math.Sin(angle)
			from := 8.2 * unit
			to := 11 * unit
			p.Line(x(12)+float32(cos)*from, y(12)+float32(sin)*from,
				x(12)+float32(cos)*to, y(12)+float32(sin)*to,
				stroke*1.1, color)
		}

	case IconClock:
		p.StrokePath(circle(x(12), y(12), 8.5*unit), stroke, color)
		p.Line(x(12), y(12), x(12), y(7), stroke, color)
		p.Line(x(12), y(12), x(16), y(14), stroke, color)

	case IconDisplay:
		p.Stroke(ui.Rect{X: x(2.5), Y: y(4), W: 19 * unit, H: 12.5 * unit}, color, 2*unit, stroke)
		p.Line(x(8), y(20.5), x(16), y(20.5), stroke, color)
		p.Line(x(12), y(16.5), x(12), y(20.5), stroke, color)

	case IconSun:
		p.StrokePath(circle(x(12), y(12), 4.6*unit), stroke, color)
		for ray := 0; ray < 8; ray++ {
			angle := float64(ray) * math.Pi / 4
			cos, sin := math.Cos(angle), math.Sin(angle)
			p.Line(x(12)+float32(cos)*7*unit, y(12)+float32(sin)*7*unit,
				x(12)+float32(cos)*9.5*unit, y(12)+float32(sin)*9.5*unit,
				stroke, color)
		}

	case IconMoon:
		path := &ui.Path{}
		path.MoveTo(x(16), y(3))
		path.CubeTo(x(8), y(4), x(5), y(11), x(8), y(17))
		path.CubeTo(x(11), y(22), x(19), y(21), x(21), y(14))
		path.CubeTo(x(15), y(17), x(11), y(10), x(16), y(3))
		path.Close()
		p.FillPath(path, color)
	}
}

func circle(cx, cy, r float32) *ui.Path {
	path := &ui.Path{}
	path.Circle(cx, cy, r)
	return path
}
