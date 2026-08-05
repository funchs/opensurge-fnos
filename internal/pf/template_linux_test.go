//go:build linux

package pf

import (
	"strings"
	"testing"

	"open-mihomo-gateway/internal/config"
)

func TestRenderAnchor(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*config.Config)
		want    []string
		notWant []string
	}{
		{
			name: "isolated lan masquerades everything from the lan",
			want: []string{
				"table inet opensurge",
				"delete table inet opensurge",
				"type nat hook postrouting priority srcnat; policy accept;",
				`oifname "en0" ip saddr 192.168.50.0/24 masquerade`,
				"type filter hook forward priority filter; policy accept;",
			},
			notWant: []string{"ip daddr !="},
		},
		{
			name: "same lan excludes local traffic",
			mutate: func(cfg *config.Config) {
				cfg.Gateway.Mode = config.GatewayModeSameLAN
				cfg.Gateway.Interface = "eth0"
				cfg.Gateway.UpstreamInterface = "eth0"
				cfg.Gateway.LANIP = "192.168.1.20"
			},
			want: []string{`oifname "eth0" ip saddr 192.168.1.0/24 ip daddr != 192.168.1.0/24 masquerade`},
		},
		{
			name: "same wifi dhcp excludes local traffic",
			mutate: func(cfg *config.Config) {
				cfg.Gateway.Mode = config.GatewayModeSameWiFiDHCP
				cfg.Gateway.Interface = "eth0"
				cfg.Gateway.UpstreamInterface = "eth0"
				cfg.Gateway.LANIP = "192.168.1.20"
			},
			want: []string{`oifname "eth0" ip saddr 192.168.1.0/24 ip daddr != 192.168.1.0/24 masquerade`},
		},
		{
			name:    "tcp redirect is never emitted",
			mutate:  func(cfg *config.Config) { cfg.PF.RedirectTCPTo = 7892 },
			notWant: []string{"redirect", "dnat"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := config.Default()
			if tc.mutate != nil {
				tc.mutate(&cfg)
			}
			rendered, err := RenderAnchor(cfg)
			if err != nil {
				t.Fatalf("RenderAnchor() error = %v", err)
			}
			for _, want := range tc.want {
				if !strings.Contains(rendered, want) {
					t.Fatalf("rendered ruleset missing %q:\n%s", want, rendered)
				}
			}
			for _, notWant := range tc.notWant {
				if strings.Contains(rendered, notWant) {
					t.Fatalf("rendered ruleset unexpectedly contains %q:\n%s", notWant, rendered)
				}
			}
		})
	}
}

// 重复 `nft -f` 不该把规则叠加两遍，靠开头的 table/delete table 惯用法保证。
func TestRenderAnchorIsIdempotentByDeletingFirst(t *testing.T) {
	rendered, err := RenderAnchor(config.Default())
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(rendered), "\n")
	if lines[0] != "table inet opensurge" || lines[1] != "delete table inet opensurge" {
		t.Fatalf("ruleset must start with the create/delete idiom, got:\n%s", rendered)
	}
}

func TestTablesContain(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   bool
	}{
		{name: "present among others", output: "table ip filter\ntable inet opensurge\ntable inet fw4\n", want: true},
		{name: "absent", output: "table ip filter\ntable inet fw4\n", want: false},
		{name: "empty ruleset", output: "", want: false},
		{name: "name is a prefix of another table", output: "table inet opensurge_old\n", want: false},
		{name: "same name in another family", output: "table ip opensurge\n", want: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tablesContain(tc.output, "opensurge"); got != tc.want {
				t.Fatalf("tablesContain() = %v, want %v", got, tc.want)
			}
		})
	}
}
