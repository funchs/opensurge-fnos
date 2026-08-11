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

// /enter 在 LAN 模式（BaseURL 已设）下直接签发 session，让浏览器直连 IP:端口可用。
func TestEnterIssuesSessionWhenBaseURLSet(t *testing.T) {
	dir := t.TempDir()
	server, err := New(Options{
		ConfigPath: filepath.Join(dir, "config.yaml"),
		Addr:       "0.0.0.0:61767",
		BaseURL:    "http://192.168.1.20:61767",
		StoreDir:   filepath.Join(dir, "store"),
		Runner:     fakeRunner{},
		Static:     http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }),
	})
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "/enter", nil)
	request.Host = "192.168.1.20:61767"
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d body=%s", recorder.Code, http.StatusFound, recorder.Body.String())
	}
	if loc := recorder.Header().Get("Location"); loc != "/dashboard" {
		t.Fatalf("Location = %q, want /dashboard", loc)
	}
	cookies := recorder.Result().Cookies()
	if len(cookies) == 0 || cookies[0].Name != "opensurge_session" || cookies[0].Value == "" {
		t.Fatalf("missing session cookie: %v", cookies)
	}
}

func TestEnterRejectedWithoutBaseURL(t *testing.T) {
	dir := t.TempDir()
	server, err := New(Options{
		ConfigPath: filepath.Join(dir, "config.yaml"),
		Addr:       "127.0.0.1:61767",
		StoreDir:   filepath.Join(dir, "store"),
		Runner:     fakeRunner{},
	})
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "/enter", nil)
	request.Host = "127.0.0.1:61767"
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
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

func TestExtraHostsAllowDomainAccess(t *testing.T) {
	dir := t.TempDir()
	server, err := New(Options{
		ConfigPath: filepath.Join(dir, "config.yaml"),
		Addr:       "0.0.0.0:61767",
		BaseURL:    "http://192.168.1.20:61767",
		ExtraHosts: []string{"opensurge.example.com", "NAS.Local"},
		StoreDir:   filepath.Join(dir, "store"),
		Runner:     fakeRunner{},
		Static:     http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }),
	})
	if err != nil {
		t.Fatal(err)
	}

	for _, host := range []string{"opensurge.example.com", "opensurge.example.com:443", "nas.local", "192.168.1.20:61767"} {
		request := httptest.NewRequest(http.MethodGet, "/", nil)
		request.Host = host
		recorder := httptest.NewRecorder()
		server.Handler().ServeHTTP(recorder, request)
		if recorder.Code != http.StatusOK {
			t.Fatalf("host %q status = %d, want 200", host, recorder.Code)
		}
	}

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Host = "evil.example"
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("evil host status = %d, want 403", recorder.Code)
	}
}

func TestLANModeAllowsIframeEmbedding(t *testing.T) {
	dir := t.TempDir()
	server, err := New(Options{
		ConfigPath: filepath.Join(dir, "config.yaml"),
		Addr:       "0.0.0.0:61767",
		BaseURL:    "http://192.168.1.20:61767",
		StoreDir:   filepath.Join(dir, "store"),
		Runner:     fakeRunner{},
		Static:     http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }),
	})
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Host = "192.168.1.20:61767"
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d", recorder.Code)
	}
	if got := recorder.Header().Get("X-Frame-Options"); got != "" {
		t.Fatalf("X-Frame-Options = %q, want empty for LAN iframe panel", got)
	}
	csp := recorder.Header().Get("Content-Security-Policy")
	if !strings.Contains(csp, "frame-ancestors *") {
		t.Fatalf("CSP = %q, want frame-ancestors * for fnOS desktop window", csp)
	}
}

func TestFnOSConnectHostAllowed(t *testing.T) {
	dir := t.TempDir()
	server, err := New(Options{
		ConfigPath: filepath.Join(dir, "config.yaml"),
		Addr:       "0.0.0.0:61767",
		BaseURL:    "http://192.168.1.20:61767",
		StoreDir:   filepath.Join(dir, "store"),
		Runner:     fakeRunner{},
		Static:     http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }),
	})
	if err != nil {
		t.Fatal(err)
	}

	// Fixed FN Connect shape: https://opensurge.<fnid>.fnos.net/
	for _, host := range []string{"opensurge.abc123.fnos.net", "opensurge.my-id.fnos.net"} {
		request := httptest.NewRequest(http.MethodGet, "/enter", nil)
		request.Host = host
		request.Header.Set("X-Forwarded-Proto", "https")
		recorder := httptest.NewRecorder()
		server.Handler().ServeHTTP(recorder, request)
		if recorder.Code != http.StatusFound {
			t.Fatalf("host %q status = %d, want 302", host, recorder.Code)
		}
		cookies := recorder.Result().Cookies()
		if len(cookies) == 0 || !cookies[0].Secure {
			t.Fatalf("host %q cookie Secure not set for HTTPS FN Connect: %v", host, cookies)
		}
	}

	for _, host := range []string{
		"evil.abc123.fnos.net",          // wrong app prefix
		"opensurge.fnos.net",            // missing fnid
		"opensurge.a.b.fnos.net",        // extra labels
		"opensurge.-bad.fnos.net",       // invalid label
		"mihomo.abc123.fnos.net",        // other app
	} {
		request := httptest.NewRequest(http.MethodGet, "/", nil)
		request.Host = host
		recorder := httptest.NewRecorder()
		server.Handler().ServeHTTP(recorder, request)
		if recorder.Code != http.StatusForbidden {
			t.Fatalf("host %q status = %d, want 403", host, recorder.Code)
		}
	}
}

func TestIsFnOSConnectHost(t *testing.T) {
	if !isFnOSConnectHost("opensurge.xxx.fnos.net") {
		t.Fatal("expected opensurge.xxx.fnos.net allowed")
	}
	if isFnOSConnectHost("opensurge.xxx.fnos.com") {
		t.Fatal("wrong TLD")
	}
}

func TestLoopbackModeDeniesIframeEmbedding(t *testing.T) {
	dir := t.TempDir()
	server, err := New(Options{
		ConfigPath: filepath.Join(dir, "config.yaml"),
		Addr:       "127.0.0.1:61767",
		StoreDir:   filepath.Join(dir, "store"),
		Runner:     fakeRunner{},
		Static:     http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }),
	})
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Host = "127.0.0.1:61767"
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, request)
	if recorder.Header().Get("X-Frame-Options") != "DENY" {
		t.Fatalf("X-Frame-Options = %q", recorder.Header().Get("X-Frame-Options"))
	}
}
