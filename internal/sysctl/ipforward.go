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
	return m.setIfNeeded("1")
}

func (m Manager) Restore(previous string) error {
	previous = strings.TrimSpace(previous)
	if previous == "" {
		return nil
	}
	return m.setIfNeeded(previous)
}

// setIfNeeded 只在当前值和目标不同时才写。容器里 /proc/sys 常是只读的
// （fnOS 连 SYS_ADMIN 都不给 remount），此时读得到、写不进去；目标值已经
// 满足就不该因为写失败而让启停失败。
func (m Manager) setIfNeeded(value string) error {
	current, err := m.Current()
	if err == nil && strings.TrimSpace(current) == value {
		return nil
	}
	return setIPForwarding(value)
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
