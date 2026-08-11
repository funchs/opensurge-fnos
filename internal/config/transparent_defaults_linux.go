//go:build linux

package config

// Linux side-router depends on mihomo pulling forwarded LAN traffic into TUN.
// auto-redirect is the Meta feature that installs nft/iptables rules for that.
func defaultTUNAutoRedirect() bool { return true }
