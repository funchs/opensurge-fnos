package controlapi

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

// 监听地址和「浏览器看到的地址」必须能分开：控制面在 NAS 上绑 0.0.0.0，
// 浏览器在另一台机器上用 http://<NAS-IP>:61767 访问。Origin 校验拿的是后者，
// 搞混了所有写操作都会 403。
func TestBaseURLSeparatesListenAddressFromBrowserOrigin(t *testing.T) {
	tests := []struct {
		name    string
		addr    string
		baseURL string
		want    string
		wantErr string
	}{
		{name: "loopback defaults to listen address", addr: "127.0.0.1:61767", want: "http://127.0.0.1:61767"},
		{name: "localhost defaults to listen address", addr: "localhost:61767", want: "http://localhost:61767"},
		{
			name:    "lan bind with explicit base url",
			addr:    "0.0.0.0:61767",
			baseURL: "http://192.168.1.20:61767",
			want:    "http://192.168.1.20:61767",
		},
		{
			name:    "trailing slash is trimmed",
			addr:    "0.0.0.0:61767",
			baseURL: "http://192.168.1.20:61767/",
			want:    "http://192.168.1.20:61767",
		},
		{
			name:    "explicit base url also applies to loopback",
			addr:    "127.0.0.1:61767",
			baseURL: "http://nas.lan:61767",
			want:    "http://nas.lan:61767",
		},
		{name: "lan bind without base url is refused", addr: "0.0.0.0:61767", wantErr: "loopback"},
		{name: "blank base url does not count", addr: "0.0.0.0:61767", baseURL: "   ", wantErr: "loopback"},
		{name: "malformed address is refused", addr: "not-an-address", wantErr: "loopback"},
		{name: "hostless base url is refused", addr: "0.0.0.0:61767", baseURL: "not a url", wantErr: "base URL is invalid"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			server, err := New(Options{
				ConfigPath: filepath.Join(dir, "config.yaml"),
				Addr:       tc.addr,
				BaseURL:    tc.baseURL,
				StoreDir:   filepath.Join(dir, "store"),
				Runner:     fakeRunner{},
			})
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("err = %v, want containing %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if server.baseURL != tc.want {
				t.Fatalf("baseURL = %q, want %q", server.baseURL, tc.want)
			}
			if !strings.HasPrefix(server.BootstrapURL(), tc.want+"/bootstrap?code=") {
				t.Fatalf("BootstrapURL() = %q, want prefix %q", server.BootstrapURL(), tc.want+"/bootstrap?code=")
			}
		})
	}
}

// Host 头校验是防 DNS rebinding 的第二道门。绑到局域网时浏览器发来的 Host 是
// NAS 的 IP，必须跟着 BaseURL 一起放行，否则连静态页都 403。
func TestAllowedHostFollowsBaseURL(t *testing.T) {
	tests := []struct {
		name     string
		baseURL  string
		host     string
		wantCode int
	}{
		{name: "loopback still allowed", baseURL: "http://192.168.1.20:61767", host: "127.0.0.1:61767", wantCode: http.StatusOK},
		{name: "localhost still allowed", baseURL: "http://192.168.1.20:61767", host: "localhost:61767", wantCode: http.StatusOK},
		{name: "base url host allowed", baseURL: "http://192.168.1.20:61767", host: "192.168.1.20:61767", wantCode: http.StatusOK},
		{name: "other host rejected", baseURL: "http://192.168.1.20:61767", host: "192.168.1.99:61767", wantCode: http.StatusForbidden},
		{name: "rebinding host rejected", baseURL: "http://192.168.1.20:61767", host: "evil.example", wantCode: http.StatusForbidden},
		{name: "lan host rejected without base url", baseURL: "", host: "192.168.1.20:61767", wantCode: http.StatusForbidden},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			server, err := New(Options{
				ConfigPath: filepath.Join(dir, "config.yaml"),
				Addr:       "127.0.0.1:61767",
				BaseURL:    tc.baseURL,
				StoreDir:   filepath.Join(dir, "store"),
				Runner:     fakeRunner{},
				Static:     http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }),
			})
			if err != nil {
				t.Fatal(err)
			}
			request := httptest.NewRequest(http.MethodGet, "/", nil)
			request.Host = tc.host
			recorder := httptest.NewRecorder()
			server.Handler().ServeHTTP(recorder, request)
			if recorder.Code != tc.wantCode {
				t.Fatalf("status = %d, want %d", recorder.Code, tc.wantCode)
			}
		})
	}
}
