//go:build linux

package macosnetwork

import (
	"context"

	"open-mihomo-gateway/internal/runtime"
)

// SystemProxy 在 Linux 上是空实现：fnOS 没有等价的「系统代理」开关，
// 而且容器改不动宿主机设置。旁路由靠设备指网关 + DNS 生效，不需要它。
type SystemProxy struct{}

func (SystemProxy) Prepare(_ context.Context, _ string, _ int) (runtime.SystemProxySnapshot, error) {
	return runtime.SystemProxySnapshot{}, ErrManagedByFnOS
}

func (SystemProxy) Enable(_ context.Context, _ runtime.SystemProxySnapshot, _ int) error {
	return ErrManagedByFnOS
}

func (SystemProxy) Restore(_ context.Context, _ runtime.SystemProxySnapshot) error {
	return ErrManagedByFnOS
}
