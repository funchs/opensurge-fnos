//go:build linux

package config

// Linux side-router depends on mihomo pulling forwarded LAN traffic into TUN.
// auto-redirect is the Meta feature that installs nft/iptables rules for that.
func defaultTUNAutoRedirect() bool { return true }

// Match conversun/fnos-apps mihomo defaults: let mihomo pick the egress NIC so
// host processes (fygo-browser Chromium, curl, etc.) share one transparent path.
func defaultTUNAutoDetectInterface() bool { return true }
