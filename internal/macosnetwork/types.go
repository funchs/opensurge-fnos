package macosnetwork

import (
	"fmt"
	"net"
	"sort"
	"strings"
)

// 这些类型和纯函数在 darwin / linux 两个平台共用，Web GUI 的 JSON 契约依赖它们。
// 平台相关的实现见 *_darwin.go / *_linux.go。

type Snapshot struct {
	NetworkService string   `json:"network_service"`
	Interface      string   `json:"interface"`
	IPv4Mode       string   `json:"-"`
	HardwareAddr   string   `json:"hardware_address,omitempty"`
	IPv4           string   `json:"ipv4,omitempty"`
	SubnetMask     string   `json:"subnet_mask,omitempty"`
	Router         string   `json:"router,omitempty"`
	DNS            []string `json:"dns"`
	IPv6Default    bool     `json:"ipv6_default"`
}

const (
	IPv4ModeDHCP   = "dhcp"
	IPv4ModeManual = "manual"
)

type ManualConfig struct {
	NetworkService string   `json:"network_service"`
	Interface      string   `json:"interface"`
	IPv4           string   `json:"ipv4"`
	SubnetMask     string   `json:"subnet_mask"`
	Router         string   `json:"router"`
	DNS            []string `json:"dns"`
}

type InterfaceOption struct {
	Interface      string `json:"interface"`
	NetworkService string `json:"network_service"`
}

// Neighbor is a currently cached IPv4-to-MAC mapping on an interface.
// The neighbor cache is observation evidence only; it is not an ownership or
// authentication guarantee.
type Neighbor struct {
	IP        string `json:"ip"`
	MAC       string `json:"mac"`
	Interface string `json:"interface"`
}

type RouteSelection struct {
	Interface string `json:"interface"`
	Gateway   string `json:"gateway,omitempty"`
}

func interfaceOptions(services map[string]string) []InterfaceOption {
	options := make([]InterfaceOption, 0, len(services))
	for service, device := range services {
		if strings.TrimSpace(service) == "" || strings.TrimSpace(device) == "" {
			continue
		}
		options = append(options, InterfaceOption{Interface: device, NetworkService: service})
	}
	sort.Slice(options, func(i, j int) bool {
		if options[i].Interface == options[j].Interface {
			return options[i].NetworkService < options[j].NetworkService
		}
		return options[i].Interface < options[j].Interface
	})
	return options
}

func ValidateManual(cfg ManualConfig) error {
	ip := net.ParseIP(cfg.IPv4).To4()
	maskIP := net.ParseIP(cfg.SubnetMask).To4()
	router := net.ParseIP(cfg.Router).To4()
	if ip == nil || maskIP == nil || router == nil {
		return fmt.Errorf("manual network configuration requires valid IPv4, subnet mask, and router")
	}
	mask := net.IPMask(maskIP)
	ones, bits := mask.Size()
	if bits != 32 || ones <= 0 || ones >= 32 {
		return fmt.Errorf("manual network configuration requires a contiguous unicast subnet mask")
	}
	if !ip.Mask(mask).Equal(router.Mask(mask)) {
		return fmt.Errorf("manual IPv4 and router must share a subnet")
	}
	if ip.Equal(router) {
		return fmt.Errorf("manual IPv4 must differ from router")
	}
	if strings.TrimSpace(cfg.NetworkService) == "" || strings.TrimSpace(cfg.Interface) == "" {
		return fmt.Errorf("network service and interface are required")
	}
	for _, server := range cfg.DNS {
		if net.ParseIP(server) == nil {
			return fmt.Errorf("invalid DNS server %q", server)
		}
	}
	return nil
}

func VerifyManual(snapshot Snapshot, expected ManualConfig) error {
	if snapshot.NetworkService != expected.NetworkService || snapshot.Interface != expected.Interface {
		return fmt.Errorf("network service or interface changed during fixed IPv4 setup")
	}
	if snapshot.IPv4Mode != IPv4ModeManual {
		if snapshot.IPv4Mode == IPv4ModeDHCP {
			return fmt.Errorf("network service %q still reports DHCP configuration", expected.NetworkService)
		}
		return fmt.Errorf("network service %q did not report manual IPv4 configuration", expected.NetworkService)
	}
	if snapshot.IPv4 != expected.IPv4 {
		return fmt.Errorf("network service %q reports IPv4 %s instead of %s", expected.NetworkService, snapshot.IPv4, expected.IPv4)
	}
	if snapshot.SubnetMask != expected.SubnetMask || snapshot.Router != expected.Router {
		return fmt.Errorf("network service %q reports an unexpected subnet mask or router", expected.NetworkService)
	}
	return nil
}
