//go:build !linux

package config

// Darwin (and other non-Linux builds) keep auto-redirect off: mihomo's
// implementation is Linux nft/iptables based and is not the Mac TUN path.
func defaultTUNAutoRedirect() bool { return false }

// Mac path pins interface via networksetup / lab defaults; keep auto-detect off.
func defaultTUNAutoDetectInterface() bool { return false }
