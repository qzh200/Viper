package widgets_test

import (
	"bytes"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"viper/internal/domain"
	"viper/internal/gui/model"
	"viper/internal/gui/theme"
	"viper/internal/gui/widgets"
)

func TestTheCardHoldsWhatItIsGiven(t *testing.T) {
	tt := ui.NewTester(func(c *ui.Context) {
		widgets.Card(c, "温度走势", func() {
			ui.Text(c, "卡片里的东西")
		})
	}, 420, 240)

	for _, want := range []string{"温度走势", "卡片里的东西"} {
		if !tt.HasText(want) {
			t.Errorf("the card has no %q in %q", want, tt.Texts())
		}
	}
}

func TestTheCardWithoutATitle(t *testing.T) {
	tt := ui.NewTester(func(c *ui.Context) {
		widgets.Card(c, "", func() {
			ui.Text(c, "只有内容")
		})
	}, 420, 240)

	if !tt.HasText("只有内容") {
		t.Errorf("the card lost its content: %q", tt.Texts())
	}
}

func TestTheIconBadgeShowsAnIcon(t *testing.T) {
	tt := ui.NewTester(func(c *ui.Context) {
		widgets.IconBadge(c, widgets.IconCPU, widgets.ColorCPU, 34)
	}, 200, 160)

	if tt.Texts() == nil && len(tt.Texts()) == 0 {
		t.Error("the icon badge drew nothing at all")
	}
}

func TestEveryIconCanBeDrawn(t *testing.T) {
	icons := []widgets.Icon{
		widgets.IconCPU, widgets.IconGPU, widgets.IconMemory,
		widgets.IconThermometer, widgets.IconFan, widgets.IconBolt,
		widgets.IconActivity, widgets.IconGear, widgets.IconClock,
		widgets.IconDisplay, widgets.IconSun, widgets.IconMoon,
	}
	for _, icon := range icons {
		tt := ui.NewTester(func(c *ui.Context) {
			widgets.IconElement(c, icon, 24, widgets.ColorCPU)
		}, 100, 100)
		if tt.Image() == nil {
			t.Errorf("the icon %d drew nothing", icon)
		}
	}
}

func TestThePillShowsItsText(t *testing.T) {
	tt := ui.NewTester(func(c *ui.Context) {
		widgets.Pill(c, "已连接", widgets.ColorGPU)
	}, 240, 120)

	if !tt.HasText("已连接") {
		t.Errorf("the pill has no text in %q", tt.Texts())
	}
}

func TestTheStatTileShowsTheReading(t *testing.T) {
	tt := ui.NewTester(func(c *ui.Context) {
		widgets.StatTile(c, widgets.IconMemory, "内存", "49.2%", "31.7 GB", widgets.ColorMemory)
	}, 320, 220)

	for _, want := range []string{"内存", "49.2%", "31.7 GB"} {
		if !tt.HasText(want) {
			t.Errorf("the tile has no %q in %q", want, tt.Texts())
		}
	}
}

func TestTheGaugeShowsTheReading(t *testing.T) {
	tt := ui.NewTester(func(c *ui.Context) {
		widgets.Gauge(c, widgets.RadialGauge{
			Value:     72,
			Low:       30,
			High:      100,
			ValueText: "72 °C",
			Label:     "温度",
			Tint:      widgets.ColorCPU,
			Size:      140,
		})
	}, 240, 240)

	for _, want := range []string{"72 °C", "温度"} {
		if !tt.HasText(want) {
			t.Errorf("the gauge has no %q in %q", want, tt.Texts())
		}
	}
}

func TestTheGaugeWithoutALabel(t *testing.T) {
	tt := ui.NewTester(func(c *ui.Context) {
		widgets.Gauge(c, widgets.RadialGauge{
			Value:     50,
			Low:       0,
			High:      100,
			ValueText: "50%",
			Tint:      widgets.ColorGPU,
		})
	}, 240, 240)

	if !tt.HasText("50%") {
		t.Errorf("the gauge has no reading in %q", tt.Texts())
	}
}

func TestTheFanIsDrawn(t *testing.T) {
	tt := ui.NewTester(func(c *ui.Context) {
		widgets.Fan(c, 2695, 90, widgets.ColorCPU)
	}, 200, 200)

	if tt.Image() == nil {
		t.Error("the fan drew nothing")
	}
}

func TestAStoppedFanStillShows(t *testing.T) {
	tt := ui.NewTester(func(c *ui.Context) {
		widgets.Fan(c, 0, 90, widgets.ColorCPU)
	}, 200, 200)

	if tt.Image() == nil {
		t.Error("the stopped fan drew nothing")
	}
}

func TestTheFanTurnsWithTime(t *testing.T) {
	tt := ui.NewTester(func(c *ui.Context) {
		widgets.Fan(c, 3000, 120, widgets.ColorCPU)
	}, 200, 200)

	before := tt.Image()
	tt.Advance(400 * time.Millisecond)
	tt.Frame()
	after := tt.Image()

	if before == nil || after == nil {
		t.Fatal("the fan drew nothing")
	}
	if bytes.Equal(before.Pix, after.Pix) {
		t.Error("the fan did not turn at all")
	}
}

func TestTheTrendNeedsSomeSamples(t *testing.T) {
	tt := ui.NewTester(func(c *ui.Context) {
		widgets.Trend(c, nil)
	}, 420, 240)

	if !tt.HasText("还在攒数据") {
		t.Errorf("the trend has no hint in %q", tt.Texts())
	}
}

func TestTheTrendDrawsTheSamples(t *testing.T) {
	history := make([]model.Sample, 30)
	for i := range history {
		history[i] = model.Sample{
			CPUTempC: 60 + i,
			GPUTempC: 55 + i/2,
			CPUUsage: float64(i),
			GPUUsage: float64(i) / 2,
		}
	}

	tt := ui.NewTester(func(c *ui.Context) {
		widgets.Trend(c, history)
	}, 640, 320)

	for _, want := range []string{"CPU", "GPU"} {
		if !tt.HasText(want) {
			t.Errorf("the legend has no %q in %q", want, tt.Texts())
		}
	}
}

func TestTheLegendShowsItsNames(t *testing.T) {
	tt := ui.NewTester(func(c *ui.Context) {
		widgets.TrendLegend(c, []string{"CPU", "GPU"}, []ui.Color{widgets.ColorCPU, widgets.ColorGPU})
	}, 320, 120)

	for _, want := range []string{"CPU", "GPU"} {
		if !tt.HasText(want) {
			t.Errorf("the legend has no %q in %q", want, tt.Texts())
		}
	}
}

func TestTheLegendSurvivesAMissingColor(t *testing.T) {
	tt := ui.NewTester(func(c *ui.Context) {
		widgets.TrendLegend(c, []string{"CPU", "GPU"}, []ui.Color{widgets.ColorCPU})
	}, 320, 120)

	if !tt.HasText("CPU") {
		t.Errorf("the legend lost its first name: %q", tt.Texts())
	}
}

func TestTheNotesAreDrawn(t *testing.T) {
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Children(func() {
			widgets.SectionTitle(c, "实时指标")
			widgets.Note(c, "还没有读到机器的数据")
			widgets.ErrorNote(c, "原厂服务没有运行")
		})
	}, 420, 240)

	for _, want := range []string{"实时指标", "还没有读到机器的数据", "原厂服务没有运行"} {
		if !tt.HasText(want) {
			t.Errorf("there is no %q in %q", want, tt.Texts())
		}
	}
}

func TestTheAppearancePickerOffersTheThreeModes(t *testing.T) {
	selected := 0
	tt := ui.NewTester(func(c *ui.Context) {
		widgets.ModePicker(c, &selected)
	}, 420, 200)

	for _, want := range []string{"自动", "亮色", "暗色"} {
		if !tt.HasText(want) {
			t.Errorf("the picker has no %q in %q", want, tt.Texts())
		}
	}
}

func TestTheAppearancePickerReportsTheChoice(t *testing.T) {
	selected := 0
	changed := false
	tt := ui.NewTester(func(c *ui.Context) {
		picker := widgets.ModePicker(c, &selected)
		if picker.Changed() {
			changed = true
		}
	}, 420, 200)

	if changed {
		t.Fatal("the picker reported a choice before any was made")
	}
	if err := tt.Click("暗色"); err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Error("the picker did not report the choice")
	}
	if selected != 2 {
		t.Errorf("the choice is %d, want the dark mode", selected)
	}
}

func TestLabelsOf(t *testing.T) {
	labels := widgets.LabelsOf([]theme.Mode{theme.ModeAuto, theme.ModeDark})
	if len(labels) != 2 || labels[0] != "自动" || labels[1] != "暗色" {
		t.Errorf("the labels are %q, want the automatic and the dark mode", labels)
	}
	if len(widgets.LabelsOf(nil)) != 0 {
		t.Error("there are labels for nothing")
	}
}

func TestTheLevelColorFollowsTheLevel(t *testing.T) {
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Text(c, "normal").TextColor(widgets.LevelColor(c, model.LevelNormal))
		ui.Text(c, "warn").TextColor(widgets.LevelColor(c, model.LevelWarn))
		ui.Text(c, "danger").TextColor(widgets.LevelColor(c, model.LevelDanger))
	}, 320, 200)

	for _, want := range []string{"normal", "warn", "danger"} {
		if !tt.HasText(want) {
			t.Errorf("there is no %q in %q", want, tt.Texts())
		}
	}
}

func TestColorWheelWidget(t *testing.T) {
	changedHue := 0.0
	changedSat := 0.0
	tt := ui.NewTester(func(c *ui.Context) {
		widgets.ColorWheel(c, widgets.ColorWheelSpec{
			Size:       140,
			Hue:        180,
			Saturation: 0.8,
			Color:      domain.RGBColor{R: 0, G: 174, B: 243},
			OnChange: func(h, s float64) {
				changedHue = h
				changedSat = s
			},
		})
	}, 200, 200)

	tt.Frame()
	if tt.Image() == nil {
		t.Error("expected color wheel image to be rendered")
	}

	tt.Press(100, 70)
	tt.Move(110, 70)
	tt.Release(110, 70)

	if changedHue == 0.0 && changedSat == 0.0 {
		t.Error("expected OnChange to be called on pointer press/drag")
	}
}
