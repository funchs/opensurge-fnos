//go:build darwin

package macosnetwork

import (
	"context"
	"fmt"
	"net"
	"strings"
)

func DiscoverNeighbors(ctx context.Context, interfaceName string) ([]Neighbor, error) {
	interfaceName = strings.TrimSpace(interfaceName)
	if interfaceName == "" {
		return nil, fmt.Errorf("interface is required")
	}
	output, err := runCommand(ctx, "/usr/sbin/arp", "-an", "-i", interfaceName)
	if err != nil {
		return nil, err
	}
	return parseNeighbors(output, interfaceName), nil
}

func parseNeighbors(output, interfaceName string) []Neighbor {
	neighbors := []Neighbor{}
	seen := map[string]bool{}
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 6 || fields[2] != "at" || fields[4] != "on" || fields[5] != interfaceName {
			continue
		}
		ipText := strings.Trim(fields[1], "()")
		ip := net.ParseIP(ipText).To4()
		mac, err := net.ParseMAC(fields[3])
		if ip == nil || err != nil || seen[ip.String()] {
			continue
		}
		seen[ip.String()] = true
		neighbors = append(neighbors, Neighbor{IP: ip.String(), MAC: strings.ToLower(mac.String()), Interface: interfaceName})
	}
	return neighbors
}
