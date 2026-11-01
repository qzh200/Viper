package probe

import (
	"fmt"

	"viper/internal/agentservice"
)

type Service interface {
	Get(function string) (agentservice.Answer, error)
	Set(function string, parameter any) (agentservice.Answer, error)
	MonitorData() (agentservice.Answer, error)
}

func decode(answer agentservice.Answer, target any) error {
	if err, failed := answer.Err(); failed {
		return err
	}
	if err := answer.Decode(target); err != nil {
		return fmt.Errorf("probe: 答案无法解析: %w", err)
	}
	return nil
}
