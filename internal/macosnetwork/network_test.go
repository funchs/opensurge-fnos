//go:build darwin

package macosnetwork

import (
	"context"
	"testing"
)

func TestParseNetworkInfo(t *testing.T) {
	got := parseNetworkInfo("DHCP Configuration\nIP address: 192.168.1.20\nSubnet mask: 255.255.255.0\nRouter: 192.168.1.1\n")
	if got.IPv4Mode != IPv4ModeDHCP || got.IPv4 != "192.168.1.20" || got.SubnetMask != "255.255.255.0" || got.Router != "192.168.1.1" {
		t.Fatalf("parseNetworkInfo() = %#v", got)
	}
	manual := parseNetworkInfo("Manual Configuration\nIP address: 192.168.1.21\nSubnet mask: 255.255.255.0\nRouter: 192.168.1.1\n")
	if manual.IPv4Mode != IPv4ModeManual {
		t.Fatalf("manual IPv4 mode = %q", manual.IPv4Mode)
	}
}

func TestParseDNSAndIPv6Default(t *testing.T) {
	dns := parseDNS("192.168.1.20\n1.1.1.1\n")
	if len(dns) != 2 {
		t.Fatalf("parseDNS() = %#v", dns)
	}
	routes := "default fe80::1%en0 UGcg en0\n::1 ::1 UHL lo0\n"
	if !hasIPv6DefaultRoute(routes, "en0") {
		t.Fatal("IPv6 default route not detected")
	}
	if hasIPv6DefaultRoute(routes, "en7") {
		t.Fatal("IPv6 default route detected on wrong interface")
	}
}

func TestParseServiceInterface(t *testing.T) {
	output := `An asterisk (*) denotes that a network service is disabled.
(1) Wi-Fi
(Hardware Port: Wi-Fi, Device: en0)
(2) Thunderbolt Bridge
(Hardware Port: Thunderbolt Bridge, Device: bridge0)
`
	device, err := parseServiceInterface(output, "Wi-Fi")
	if err != nil {
		t.Fatal(err)
	}
	if device != "en0" {
		t.Fatalf("device = %q", device)
	}
	if _, err := parseServiceInterface(output, "Missing"); err == nil {
		t.Fatal("missing service should fail")
	}
	if got := parseServiceOrder(output)["Thunderbolt Bridge"]; got != "bridge0" {
		t.Fatalf("bridge service = %q", got)
	}
}

func TestLookupRouteReturnsSelectedInterface(t *testing.T) {
	original := runCommand
	t.Cleanup(func() { runCommand = original })
	runCommand = func(_ context.Context, binary string, args ...string) (string, error) {
		if binary != "/sbin/route" || len(args) != 3 {
			t.Fatalf("command = %s %#v", binary, args)
		}
		return "   gateway: 198.18.0.1\n interface: utun42\n", nil
	}

	route, err := LookupRoute(t.Context(), "1.0.0.0")
	if err != nil {
		t.Fatal(err)
	}
	if route.Interface != "utun42" || route.Gateway != "198.18.0.1" {
		t.Fatalf("route = %#v", route)
	}
}

func TestParseRouteGet(t *testing.T) {
	got := parseRouteGet("   route to: 1.1.1.1\n    gateway: 198.18.0.1\n  interface: utun123\n")
	if got.Interface != "utun123" || got.Gateway != "198.18.0.1" {
		t.Fatalf("parseRouteGet() = %#v", got)
	}
}
