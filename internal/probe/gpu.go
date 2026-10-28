package probe

import (
	"encoding/json"
	"strings"
)

// MonitorValue returns one value out of the monitor data.
//
// 监视数据是一张平铺的表，按名字取其中一项。
func MonitorValue(data json.RawMessage, name string) (any, bool) {
	var document map[string]any
	if err := json.Unmarshal(data, &document); err != nil {
		return nil, false
	}
	value, ok := document[name]
	return value, ok
}

// GraphicsLines returns the lines of the monitor data that carry the graphics
// card.
//
// 显卡的那几项在监视数据里是混在一起的，读的时候按名字挑出来。
func GraphicsLines(data json.RawMessage) []string {
	var document any
	if err := json.Unmarshal(data, &document); err != nil {
		return nil
	}

	text, err := json.MarshalIndent(document, "  ", "  ")
	if err != nil {
		return nil
	}

	var out []string
	for _, line := range strings.Split(string(text), "\n") {
		lower := strings.ToLower(line)
		if strings.Contains(lower, "gpu") || strings.Contains(lower, "frequency") ||
			strings.Contains(lower, "usage") || strings.Contains(lower, "idle") ||
			strings.Contains(lower, "clock") {
			out = append(out, strings.TrimSpace(line))
		}
	}
	return out
}
