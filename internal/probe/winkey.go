package probe

import "viper/internal/agentservice"

// WinKeyStatus is the state of the Windows key lock.
//
// 编号 1 表示这个键还能用，所以锁住的时候是 0。
type WinKeyStatus struct {
	Status int `json:"status"`
}

// ReadWinKey asks the service about the Windows key.
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

// WinKeyPayload builds the write of the Windows key lock.
func WinKeyPayload(status int) map[string]any {
	return map[string]any{"status": status}
}

// WriteWinKey writes the Windows key lock.
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
