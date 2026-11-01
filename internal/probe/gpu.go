package probe

import (
	"encoding/json"
	"strings"
)

func MonitorValue(data json.RawMessage, name string) (any, bool) {
	var document map[string]any
	if err := json.Unmarshal(data, &document); err != nil {
		return nil, false
	}
	value, ok := document[name]
	return value, ok
}

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
