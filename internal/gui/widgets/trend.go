package widgets

import (
	"fmt"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/chart"

	"viper/internal/gui/model"
)

func Trend(c *ui.Context, history []model.Sample) ui.Element {
	t := c.Theme()

	if len(history) < 2 {
		return ui.Column(c).Gap(8).Children(func() {
			ui.Text(c, "还在攒数据，过一会儿就有曲线了。").FontSize(12).TextColor(t.TextMuted)
		})
	}

	cpuTemp, gpuTemp, boardTemp := make([]float64, len(history)), make([]float64, len(history)), make([]float64, len(history))
	hi := 60.0
	for i, sample := range history {
		cpuTemp[i] = float64(sample.CPUTempC)
		gpuTemp[i] = float64(sample.GPUTempC)
		boardTemp[i] = float64(sample.BoardTempC)
		for _, value := range []float64{cpuTemp[i], gpuTemp[i], boardTemp[i]} {
			if value > hi {
				hi = value
			}
		}
	}

	first, last := history[0].Time, history[len(history)-1].Time
	x := chart.NewTime(first, last)
	y := chart.NewLinear(0, hi).Nice(4)
	y.Format = func(v float64) string { return fmt.Sprintf("%.0f°", v) }
	yt := y.Ticks(4)
	xt := secondTicks(x, first, last)

	xs := make([]float32, len(history))
	for i := range xs {
		xs[i] = x.At(history[i].Time)
	}
	cpuYs := make([]float32, len(history))
	gpuYs := make([]float32, len(history))
	boardYs := make([]float32, len(history))
	for i := range history {
		cpuYs[i] = y.At(cpuTemp[i])
		gpuYs[i] = y.At(gpuTemp[i])
		boardYs[i] = y.At(boardTemp[i])
	}

	return ui.Column(c).Gap(10).Grow(1).Children(func() {
		chart.Plot(c, chart.Axes{Left: yt, Bottom: xt, Label: trendAxisLabel(c)}, func(area ui.Element) {
			area.Draw(func(p *ui.Painter, r ui.Rect) {
				for _, tick := range xt {
					x := chart.X(r, tick.At)
					p.Line(x, r.Y, x, r.Y+r.H, 1, t.Border)
				}
				for _, tick := range yt {
					p.Line(r.X, chart.Y(r, tick.At), r.X+r.W, chart.Y(r, tick.At), 1, t.Border)
				}
				p.FillPath(chart.Area(r, xs, nil, cpuYs, chart.CurveMonotone), ColorCPU.Alpha(0.10))
				p.FillPath(chart.Area(r, xs, nil, gpuYs, chart.CurveMonotone), ColorGPU.Alpha(0.10))
				p.StrokePath(chart.Line(r, xs, cpuYs, chart.CurveMonotone), 2, ColorCPU)
				p.StrokePath(chart.Line(r, xs, gpuYs, chart.CurveMonotone), 2, ColorGPU)
				p.StrokePath(chart.Line(r, xs, boardYs, chart.CurveMonotone), 2, ColorBoard)
			})
		}).Label("温度走势").Grow(1).MinHeight(120).GapX(10).GapY(6)

		TrendLegend(c, []string{"CPU", "GPU", "主板"},
			[]ui.Color{ColorCPU, ColorGPU, ColorBoard})
	})
}

func secondTicks(scale chart.Time, first, last time.Time) []chart.Tick {
	span := last.Sub(first)
	if span <= 0 {
		return nil
	}

	step := time.Minute
	for _, candidate := range []time.Duration{
		2 * time.Second, 5 * time.Second, 10 * time.Second, 15 * time.Second,
		30 * time.Second, time.Minute,
	} {
		step = candidate
		if span/candidate <= 10 {
			break
		}
	}

	start := first.Truncate(step)
	var ticks []chart.Tick
	for moment := start; !moment.After(last); moment = moment.Add(step) {
		if moment.Before(first) {
			continue
		}
		ticks = append(ticks, chart.Tick{
			At:    scale.At(moment),
			Value: float64(moment.Unix()),
			Label: moment.Format("15:04:05"),
		})
	}
	return ticks
}

func TrendLegend(c *ui.Context, names []string, colors []ui.Color) {
	t := c.Theme()
	ui.Row(c).Gap(16).Children(func() {
		for i, name := range names {
			if i >= len(colors) {
				break
			}
			ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
				ui.Box(c).Size(9, 9).Radius(3).Background(colors[i])
				ui.Text(c, name).FontSize(12).TextColor(t.TextMuted)
			})
		}
	})
}

func trendAxisLabel(c *ui.Context) func(chart.Side, chart.Tick) {
	return func(_ chart.Side, tick chart.Tick) {
		ui.Text(c, tick.Label).NoWrap().FontSize(11).TextColor(c.Theme().TextMuted)
	}
}
