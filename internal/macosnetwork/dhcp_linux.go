//go:build linux

package macosnetwork

import (
	"context"
	"time"
)

// 旁路由模式不接管 DHCP，探测 DHCP 服务器只服务于「改本机网卡」流程。
func ProbeDHCPServers(_ context.Context, _ string, _ time.Duration) ([]string, error) {
	return nil, ErrManagedByFnOS
}
