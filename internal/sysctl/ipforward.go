package sysctl

import (
	"fmt"
	"os/exec"
	"strings"

	"open-mihomo-gateway/internal/process"
)

type Manager struct{}

func New() Manager {
	return Manager{}
}

func (m Manager) Check() error {
	if _, err := exec.LookPath("sysctl"); err != nil {
		return fmt.Errorf("sysctl not found in PATH")
	}
	return nil
}

func (m Manager) Current() (string, error) {
	out, err := process.Output("sysctl", "-n", keyIPForwarding)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func (m Manager) Enable() error {
	// 先检查当前值，已经是 1 就不写（容器环境下 /proc/sys 可能只读）
	current, err := m.Current()
	if err == nil && strings.TrimSpace(current) == "1" {
		return nil
	}
	return setIPForwarding("1")
}

func (m Manager) Restore(previous string) error {
	previous = strings.TrimSpace(previous)
	if previous == "" {
		return nil
	}
	return setIPForwarding(previous)
}

func setIPForwarding(value string) error {
	return process.Run("sysctl", "-w", keyIPForwarding+"="+value)
}

func FormatForwarding(value string) string {
	switch strings.TrimSpace(value) {
	case "1":
		return "enabled"
	case "0":
		return "disabled"
	default:
		return "unknown"
	}
}
