package widgets

import (
	"image"
	"image/color"
	"math"
	"sync"

	"github.com/egoist/mygo/ui"

	"viper/internal/domain"
	"viper/internal/gui/model"
)

var (
	wheelOnce   sync.Once
	wheelBitmap *ui.Bitmap
)

// getSmoothWheelBitmap 返回全局缓存的超高分辨率 360 度逐像素平滑抗锯齿 HSV 色盘位图。
func getSmoothWheelBitmap() *ui.Bitmap {
	wheelOnce.Do(func() {
		const size = 320
		const cx = float64(size) / 2.0
		const cy = float64(size) / 2.0
		const radius = float64(size)/2.0 - 4.0

		img := image.NewRGBA(image.Rect(0, 0, size, size))
		for y := 0; y < size; y++ {
			for x := 0; x < size; x++ {
				dx := float64(x) + 0.5 - cx
				dy := float64(y) + 0.5 - cy
				dist := math.Hypot(dx, dy)
				if dist > radius {
					continue
				}

				// 边缘亚像素平滑抗锯齿
				alpha := 1.0
				if dist > radius-1.2 {
					alpha = (radius - dist) / 1.2
					if alpha < 0 {
						alpha = 0
					} else if alpha > 1 {
						alpha = 1
					}
				}

				sat := dist / (radius - 1.0)
				if sat > 1.0 {
					sat = 1.0
				}
				angle := math.Atan2(dy, dx) * 180.0 / math.Pi
				if angle < 0 {
					angle += 360.0
				}

				rgb := model.HSVToRGB(angle, sat, 1.0)
				a := uint8(math.Round(255 * alpha))
				r := uint8(math.Round(float64(rgb.R) * alpha))
				g := uint8(math.Round(float64(rgb.G) * alpha))
				b := uint8(math.Round(float64(rgb.B) * alpha))
				img.SetRGBA(x, y, color.RGBA{R: r, G: g, B: b, A: a})
			}
		}
		wheelBitmap = ui.NewBitmap(img)
	})
	return wheelBitmap
}

// ColorWheelSpec 描述色轮组件的渲染与交互属性。
type ColorWheelSpec struct {
	Size       float32
	Hue        float64
	Saturation float64
	Color      domain.RGBColor
	OnChange   func(hue float64, sat float64)
}

// ColorWheel 绘制原生 PredatorSense 风格的极坐标 360 度平滑色轮，
// 并完整支持鼠标点击与按住实时拖动取色。
func ColorWheel(c *ui.Context, spec ColorWheelSpec) ui.Element {
	size := spec.Size
	if size <= 0 {
		size = 148
	}

	box := ui.Box(c).Size(size, size).Cursor(ui.CursorCrosshair)

	// 指针交互处理：支持点击任意点以及按住鼠标实时拖拽取色
	if spec.OnChange != nil {
		px, py, over := box.PointerPosition()
		if box.Pressed() || (over && box.Clicked()) {
			cx := size / 2
			cy := size / 2
			dx := float64(px - cx)
			dy := float64(py - cy)
			dist := math.Hypot(dx, dy)
			radius := float64(size/2 - 3)
			if dist > 0 {
				angle := math.Atan2(dy, dx) * 180.0 / math.Pi
				if angle < 0 {
					angle += 360.0
				}
				sat := dist / radius
				if sat > 1.0 {
					sat = 1.0
				}
				spec.OnChange(angle, sat)
			}
		}
	}

	return box.Draw(func(p *ui.Painter, r ui.Rect) {
		t := c.Theme()
		cx, cy := r.X+r.W/2, r.Y+r.H/2
		radius := min(r.W, r.H)/2 - 3

		// 1. 绘制逐像素平滑渲染的全局色轮位图
		bmp := getSmoothWheelBitmap()
		p.Image(bmp, ui.Rect{
			X: cx - radius,
			Y: cy - radius,
			W: radius * 2,
			H: radius * 2,
		}, ui.Contain)

		// 2. 外边框描边
		p.StrokePath(circle(cx, cy, radius), 1.5, t.Border)

		// 3. 当前颜色定位光标 (Picker Handle)
		sat := spec.Saturation
		if sat <= 0 {
			sat = 1.0
		}
		if sat > 1.0 {
			sat = 1.0
		}
		dist := radius * float32(sat)
		angleRad := spec.Hue * math.Pi / 180
		hx := cx + dist*float32(math.Cos(angleRad))
		hy := cy + dist*float32(math.Sin(angleRad))

		// 双层白环黑影光标，并填充当前选中的颜色
		p.StrokePath(circle(hx, hy, 8), 1.5, ui.RGBA(0, 0, 0, 0.45))
		p.StrokePath(circle(hx, hy, 7), 2.0, ui.RGB(255, 255, 255))
		p.FillPath(circle(hx, hy, 4.5), ui.RGB(spec.Color.R, spec.Color.G, spec.Color.B))
	})
}
