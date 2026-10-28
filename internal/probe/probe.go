// Package probe holds the protocol knowledge that the forensic tools use.
//
// The tools themselves are thin: they read the command line, call this
// package and print what comes back. The knowledge lives here so that it can
// be checked by tests without a machine.
//
// 这个包装着那些取证工具用到的协议知识。工具本身是薄的：读命令行、调用这里
// 的函数、把结果打出来。知识放在这里，所以不用真的机器也能有用例守着。
package probe

import (
	"fmt"

	"viper/internal/agentservice"
)

// Service is the part of the original service that the probes use.
//
// The client of the service satisfies it, and a test can put a stand in its
// place.
type Service interface {
	// Get asks for the value of a setting.
	Get(function string) (agentservice.Answer, error)
	// Set writes a setting.
	Set(function string, parameter any) (agentservice.Answer, error)
	// MonitorData asks for the live numbers of the machine.
	MonitorData() (agentservice.Answer, error)
}

// decode reads the payload of an answer into a value.
func decode(answer agentservice.Answer, target any) error {
	if err, failed := answer.Err(); failed {
		return err
	}
	if err := answer.Decode(target); err != nil {
		return fmt.Errorf("probe: 答案无法解析: %w", err)
	}
	return nil
}
