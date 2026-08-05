//go:build linux

package macosnetwork

import (
	"context"
	"fmt"
)

func LookupRoute(ctx context.Context, destination string) (RouteSelection, error) {
	output, err := runCommand(ctx, "ip", "-j", "route", "get", destination)
	if err != nil {
		return RouteSelection{}, err
	}
	route := parseRouteGet(output)
	if route.Interface == "" {
		return RouteSelection{}, fmt.Errorf("route lookup for %s did not report an interface", destination)
	}
	return route, nil
}

// parseRouteGet 读取 `ip -j route get <dst>` 的输出，取首条结果的 dev / gateway。
func parseRouteGet(output string) RouteSelection {
	for _, entry := range parseIPRoutes(output) {
		if entry.Dev == "" {
			continue
		}
		return RouteSelection{Interface: entry.Dev, Gateway: entry.Gateway}
	}
	return RouteSelection{}
}
