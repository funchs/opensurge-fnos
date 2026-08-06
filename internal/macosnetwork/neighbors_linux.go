//go:build linux

package macosnetwork

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strings"
)

func DiscoverNeighbors(ctx context.Context, interfaceName string) ([]Neighbor, error) {
	interfaceName = strings.TrimSpace(interfaceName)
	if interfaceName == "" {
		return nil, fmt.Errorf("interface is required")
	}
	output, err := runCommand(ctx, "ip", "-j", "neigh", "show", "dev", interfaceName)
	if err != nil {
		return nil, err
	}
	return parseNeighbors(output, interfaceName)
}

type ipNeighEntry struct {
	Dst    string   `json:"dst"`
	LLAddr string   `json:"lladdr"`
	State  []string `json:"state"`
}

// usableNeighborStates 只保留有可信 MAC 的状态；FAILED / INCOMPLETE / NOARP 排除。
var usableNeighborStates = map[string]bool{
	"REACHABLE": true,
	"STALE":     true,
	"DELAY":     true,
	"PROBE":     true,
	"PERMANENT": true,
}

func parseNeighbors(output, interfaceName string) ([]Neighbor, error) {
	var entries []ipNeighEntry
	if err := json.Unmarshal([]byte(output), &entries); err != nil {
		return nil, fmt.Errorf("parse ip neigh output: %w", err)
	}
	neighbors := []Neighbor{}
	seen := map[string]bool{}
	for _, entry := range entries {
		if !hasUsableState(entry.State) {
			continue
		}
		ip := net.ParseIP(entry.Dst).To4()
		mac, err := net.ParseMAC(entry.LLAddr)
		if ip == nil || err != nil || seen[ip.String()] {
			continue
		}
		seen[ip.String()] = true
		neighbors = append(neighbors, Neighbor{IP: ip.String(), MAC: strings.ToLower(mac.String()), Interface: interfaceName})
	}
	return neighbors, nil
}

func hasUsableState(states []string) bool {
	for _, state := range states {
		if usableNeighborStates[strings.ToUpper(state)] {
			return true
		}
	}
	return false
}
