//go:build linux

package doctor

// Linux 上防火墙走 nftables（table inet opensurge），见 internal/pf/manager_linux.go。
const (
	firewallCheckName = "nft"
	firewallCommand   = "nft"
)
