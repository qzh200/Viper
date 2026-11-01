package probe

import "viper/internal/agentservice"

type WinKeyStatus struct {
	Status int `json:"status"`
}

func ReadWinKey(service Service) (WinKeyStatus, error) {
	answer, err := service.Get(agentservice.FunctionWinKey)
	if err != nil {
		return WinKeyStatus{}, err
	}

	var status WinKeyStatus
	if err := decode(answer, &status); err != nil {
		return WinKeyStatus{}, err
	}
	return status, nil
}

func WinKeyPayload(status int) map[string]any {
	return map[string]any{"status": status}
}

func WriteWinKey(service Service, status int) error {
	answer, err := service.Set(agentservice.FunctionWinKey, WinKeyPayload(status))
	if err != nil {
		return err
	}
	if err, failed := answer.Err(); failed {
		return err
	}
	return nil
}
