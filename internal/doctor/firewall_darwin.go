//go:build darwin

package doctor

// macOS 的防火墙走 pf anchor，见 internal/pf/manager.go。
const (
	firewallCheckName = "pfctl"
	firewallCommand   = "pfctl"
)
