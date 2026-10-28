package probe

import "viper/internal/agentservice"

// WinKeyStatus 表示 Win 键启用/锁定的状态码。
type WinKeyStatus struct {
	Status int `json:"status"`
}

// ReadWinKey 读取当前的 Win 键状态。
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

// WinKeyPayload 构造 Win 键状态写入请求体。
func WinKeyPayload(status int) map[string]any {
	return map[string]any{"status": status}
}

// WriteWinKey 向服务写入指定的 Win 键状态。
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
