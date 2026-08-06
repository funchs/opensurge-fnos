//go:build linux

package macosnetwork

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseNeighbors(t *testing.T) {
	tests := []struct {
		name    string
		output  string
		want    []Neighbor
		wantErr string
	}{
		{
			name: "keeps reachable and stale, drops failed",
			output: `[{"dst":"192.168.1.1","dev":"eth0","lladdr":"AA:BB:CC:DD:EE:FF","state":["REACHABLE"]},` +
				`{"dst":"192.168.1.5","dev":"eth0","lladdr":"11:22:33:44:55:66","state":["STALE"]},` +
				`{"dst":"192.168.1.9","dev":"eth0","state":["FAILED"]}]`,
			want: []Neighbor{
				{IP: "192.168.1.1", MAC: "aa:bb:cc:dd:ee:ff", Interface: "eth0"},
				{IP: "192.168.1.5", MAC: "11:22:33:44:55:66", Interface: "eth0"},
			},
		},
		{
			name:   "drops ipv6 neighbors",
			output: `[{"dst":"fe80::1","dev":"eth0","lladdr":"aa:bb:cc:dd:ee:ff","router":true,"state":["STALE"]}]`,
			want:   []Neighbor{},
		},
		{
			name:   "drops incomplete entries without lladdr",
			output: `[{"dst":"192.168.1.7","dev":"eth0","state":["INCOMPLETE"]}]`,
			want:   []Neighbor{},
		},
		{
			name: "dedupes repeated ip",
			output: `[{"dst":"192.168.1.1","dev":"eth0","lladdr":"aa:bb:cc:dd:ee:ff","state":["REACHABLE"]},` +
				`{"dst":"192.168.1.1","dev":"eth0","lladdr":"11:22:33:44:55:66","state":["STALE"]}]`,
			want: []Neighbor{{IP: "192.168.1.1", MAC: "aa:bb:cc:dd:ee:ff", Interface: "eth0"}},
		},
		{
			name:   "keeps permanent entries",
			output: `[{"dst":"192.168.1.2","dev":"eth0","lladdr":"aa:bb:cc:dd:ee:01","state":["PERMANENT"]}]`,
			want:   []Neighbor{{IP: "192.168.1.2", MAC: "aa:bb:cc:dd:ee:01", Interface: "eth0"}},
		},
		{name: "empty cache", output: `[]`, want: []Neighbor{}},
		{name: "not json", output: "192.168.1.1 dev eth0 lladdr aa:bb:cc:dd:ee:ff REACHABLE", wantErr: "parse ip neigh output"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseNeighbors(tc.output, "eth0")
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("err = %v, want containing %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("parseNeighbors() = %#v, want %#v", got, tc.want)
			}
		})
	}
}
