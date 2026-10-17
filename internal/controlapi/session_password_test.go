package controlapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const lanOrigin = "http://192.168.1.20:61767"

func newLANControl(t *testing.T) *Server {
	t.Helper()
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
	return server
}

func lanJSON(server *Server, method, path, origin, body, remote string, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Host = "192.168.1.20:61767"
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	req.RemoteAddr = remote
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	return rec
}

func sessionCookie(rec *httptest.ResponseRecorder) *http.Cookie {
	var found *http.Cookie
	for _, cookie := range rec.Result().Cookies() {
		if cookie.Name == "opensurge_session" {
			c := cookie
			found = c
		}
	}
	return found
}

func apiErrorCode(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var payload ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode error: %v body=%s", err, rec.Body.String())
	}
	return payload.Error.Code
}

func TestSessionStatusBeforeAndAfterSetup(t *testing.T) {
	server := newLANControl(t)
	rec := lanJSON(server, http.MethodGet, "/api/v1/session/status", "", "", "192.168.1.50:9")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var before sessionStatusResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &before); err != nil {
		t.Fatal(err)
	}
	if !before.GUIPasswordRequired || before.PasswordConfigured || before.Authenticated {
		t.Fatalf("before = %+v", before)
	}

	rec = lanJSON(server, http.MethodPost, "/api/v1/session/setup", lanOrigin, `{"password":"long-enough-password"}`, "192.168.1.50:9")
	if rec.Code != http.StatusNoContent {
		t.Fatalf("setup status = %d body=%s", rec.Code, rec.Body.String())
	}
	cookie := sessionCookie(rec)
	if cookie == nil || cookie.Value == "" {
		t.Fatal("setup did not issue session")
	}
	info, err := os.Stat(filepath.Join(server.store.Dir(), "gui-password"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("perm = %o", info.Mode().Perm())
	}

	rec = lanJSON(server, http.MethodGet, "/api/v1/session/status", "", "", "192.168.1.50:9", cookie)
	var after sessionStatusResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &after); err != nil {
		t.Fatal(err)
	}
	if !after.PasswordConfigured || !after.Authenticated {
		t.Fatalf("after = %+v", after)
	}
}

func TestSessionSetupRejectsDuplicateMissingOriginAndShortPassword(t *testing.T) {
	server := newLANControl(t)
	rec := lanJSON(server, http.MethodPost, "/api/v1/session/setup", lanOrigin, `{"password":"long-enough-password"}`, "192.168.1.50:9")
	if rec.Code != http.StatusNoContent {
		t.Fatalf("first setup = %d %s", rec.Code, rec.Body.String())
	}
	rec = lanJSON(server, http.MethodPost, "/api/v1/session/setup", lanOrigin, `{"password":"another-long-password"}`, "192.168.1.50:9")
	if rec.Code != http.StatusConflict || apiErrorCode(t, rec) != "password_already_configured" {
		t.Fatalf("duplicate = %d %s", rec.Code, rec.Body.String())
	}

	server = newLANControl(t)
	rec = lanJSON(server, http.MethodPost, "/api/v1/session/setup", "", `{"password":"long-enough-password"}`, "192.168.1.50:9")
	if rec.Code != http.StatusForbidden || apiErrorCode(t, rec) != "origin_rejected" {
		t.Fatalf("missing origin = %d %s", rec.Code, rec.Body.String())
	}

	rec = lanJSON(server, http.MethodPost, "/api/v1/session/setup", lanOrigin, `{"password":"short"}`, "192.168.1.50:9")
	if rec.Code != http.StatusBadRequest || apiErrorCode(t, rec) != "invalid_request" {
		t.Fatalf("short = %d %s", rec.Code, rec.Body.String())
	}
}

func TestSessionLoginSuccessAndErrors(t *testing.T) {
	server := newLANControl(t)
	rec := lanJSON(server, http.MethodPost, "/api/v1/session/login", lanOrigin, `{"password":"long-enough-password"}`, "192.168.1.50:9")
	if rec.Code != http.StatusConflict || apiErrorCode(t, rec) != "password_not_configured" {
		t.Fatalf("login before setup = %d %s", rec.Code, rec.Body.String())
	}
	if rec := lanJSON(server, http.MethodPost, "/api/v1/session/setup", lanOrigin, `{"password":"long-enough-password"}`, "192.168.1.50:9"); rec.Code != http.StatusNoContent {
		t.Fatalf("setup = %d %s", rec.Code, rec.Body.String())
	}
	rec = lanJSON(server, http.MethodPost, "/api/v1/session/login", lanOrigin, `{"password":"wrong-password-xx"}`, "192.168.1.50:9")
	if rec.Code != http.StatusUnauthorized || apiErrorCode(t, rec) != "invalid_password" {
		t.Fatalf("wrong = %d %s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "wrong-password-xx") {
		t.Fatalf("password leaked in error body: %s", rec.Body.String())
	}
	rec = lanJSON(server, http.MethodPost, "/api/v1/session/login", lanOrigin, `{"password":"long-enough-password"}`, "192.168.1.50:9")
	if rec.Code != http.StatusNoContent || sessionCookie(rec) == nil {
		t.Fatalf("login = %d %s cookies=%v", rec.Code, rec.Body.String(), rec.Result().Cookies())
	}
}

func TestSessionLoginRateLimit(t *testing.T) {
	server := newLANControl(t)
	if rec := lanJSON(server, http.MethodPost, "/api/v1/session/setup", lanOrigin, `{"password":"long-enough-password"}`, "192.168.1.50:9"); rec.Code != http.StatusNoContent {
		t.Fatal(rec.Body.String())
	}
	for i := 0; i < 5; i++ {
		rec := lanJSON(server, http.MethodPost, "/api/v1/session/login", lanOrigin, `{"password":"wrong-password-xx"}`, "192.168.1.50:9")
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("attempt %d status=%d", i, rec.Code)
		}
	}
	rec := lanJSON(server, http.MethodPost, "/api/v1/session/login", lanOrigin, `{"password":"wrong-password-xx"}`, "192.168.1.50:9")
	if rec.Code != http.StatusTooManyRequests || apiErrorCode(t, rec) != "too_many_attempts" {
		t.Fatalf("limited = %d %s", rec.Code, rec.Body.String())
	}
	other := lanJSON(server, http.MethodPost, "/api/v1/session/login", lanOrigin, `{"password":"wrong-password-xx"}`, "192.168.1.51:9")
	if other.Code != http.StatusUnauthorized {
		t.Fatalf("other IP should not share budget: %d %s", other.Code, other.Body.String())
	}
	ok := lanJSON(server, http.MethodPost, "/api/v1/session/login", lanOrigin, `{"password":"long-enough-password"}`, "192.168.1.51:9")
	if ok.Code != http.StatusNoContent {
		t.Fatalf("success on other IP = %d %s", ok.Code, ok.Body.String())
	}
}

func TestSessionLoginRateLimitUsesForwardedForOnFNConnect(t *testing.T) {
	server := newLANControl(t)
	if rec := lanJSON(server, http.MethodPost, "/api/v1/session/setup", lanOrigin, `{"password":"long-enough-password"}`, "192.168.1.50:9"); rec.Code != http.StatusNoContent {
		t.Fatal(rec.Body.String())
	}
	login := func(xff string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/session/login", strings.NewReader(`{"password":"wrong-password-xx"}`))
		req.Host = "opensurge.abc123.fnos.net"
		req.Header.Set("Origin", "https://opensurge.abc123.fnos.net")
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Forwarded-Proto", "https")
		req.Header.Set("X-Forwarded-For", xff)
		req.RemoteAddr = "10.0.0.1:9"
		rec := httptest.NewRecorder()
		server.Handler().ServeHTTP(rec, req)
		return rec
	}
	for i := 0; i < 5; i++ {
		if rec := login("203.0.113.10"); rec.Code != http.StatusUnauthorized {
			t.Fatalf("attempt %d status=%d %s", i, rec.Code, rec.Body.String())
		}
	}
	if rec := login("203.0.113.10"); rec.Code != http.StatusTooManyRequests {
		t.Fatalf("xff limited = %d %s", rec.Code, rec.Body.String())
	}
	if rec := login("203.0.113.11"); rec.Code != http.StatusUnauthorized {
		t.Fatalf("other xff should not share budget: %d %s", rec.Code, rec.Body.String())
	}
}

func TestEnterRedirectsWithoutGrantingCookie(t *testing.T) {
	server := newLANControl(t)
	req := httptest.NewRequest(http.MethodGet, "/enter", nil)
	req.Host = "192.168.1.20:61767"
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusFound || rec.Header().Get("Location") != "/setup" {
		t.Fatalf("unset enter = %d loc=%q", rec.Code, rec.Header().Get("Location"))
	}
	if sessionCookie(rec) != nil {
		t.Fatal("enter must not issue cookie before password is set")
	}

	if rec := lanJSON(server, http.MethodPost, "/api/v1/session/setup", lanOrigin, `{"password":"long-enough-password"}`, "192.168.1.50:9"); rec.Code != http.StatusNoContent {
		t.Fatal(rec.Body.String())
	}
	req = httptest.NewRequest(http.MethodGet, "/enter", nil)
	req.Host = "192.168.1.20:61767"
	rec = httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusFound || rec.Header().Get("Location") != "/login" {
		t.Fatalf("configured enter = %d loc=%q", rec.Code, rec.Header().Get("Location"))
	}
	if sessionCookie(rec) != nil {
		t.Fatal("enter must not issue cookie when unauthenticated")
	}

	login := lanJSON(server, http.MethodPost, "/api/v1/session/login", lanOrigin, `{"password":"long-enough-password"}`, "192.168.1.50:9")
	cookie := sessionCookie(login)
	if cookie == nil {
		t.Fatal("login cookie")
	}
	req = httptest.NewRequest(http.MethodGet, "/enter", nil)
	req.Host = "192.168.1.20:61767"
	req.AddCookie(cookie)
	rec = httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusFound || rec.Header().Get("Location") != "/dashboard" {
		t.Fatalf("authed enter = %d loc=%q", rec.Code, rec.Header().Get("Location"))
	}
	if sessionCookie(rec) == nil {
		t.Fatal("authed enter should refresh cookie")
	}
}

func TestEnterRejectedWithoutBaseURLStill(t *testing.T) {
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
	req := httptest.NewRequest(http.MethodPost, "/api/v1/session/setup", strings.NewReader(`{"password":"long-enough-password"}`))
	req.Host = "127.0.0.1:61767"
	req.Header.Set("Origin", "http://127.0.0.1:61767")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("loopback setup = %d %s", rec.Code, rec.Body.String())
	}
}

func TestDeletedPasswordFileInvalidatesSession(t *testing.T) {
	server := newLANControl(t)
	rec := lanJSON(server, http.MethodPost, "/api/v1/session/setup", lanOrigin, `{"password":"long-enough-password"}`, "192.168.1.50:9")
	cookie := sessionCookie(rec)
	if cookie == nil {
		t.Fatal("cookie")
	}
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.AddCookie(cookie)
	authed := httptest.NewRecorder()
	server.auth(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })).ServeHTTP(authed, req)
	if authed.Code != http.StatusNoContent {
		t.Fatalf("before delete = %d", authed.Code)
	}
	if err := os.Remove(filepath.Join(server.store.Dir(), "gui-password")); err != nil {
		t.Fatal(err)
	}
	authed = httptest.NewRecorder()
	server.auth(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })).ServeHTTP(authed, req)
	if authed.Code != http.StatusUnauthorized {
		t.Fatalf("after delete = %d body=%s", authed.Code, authed.Body.String())
	}
}

func TestChangePasswordInvalidatesOtherSessions(t *testing.T) {
	server := newLANControl(t)
	first := lanJSON(server, http.MethodPost, "/api/v1/session/setup", lanOrigin, `{"password":"old-password-ok"}`, "192.168.1.50:9")
	oldCookie := sessionCookie(first)
	second := lanJSON(server, http.MethodPost, "/api/v1/session/login", lanOrigin, `{"password":"old-password-ok"}`, "192.168.1.51:9")
	otherCookie := sessionCookie(second)
	if oldCookie == nil || otherCookie == nil {
		t.Fatal("cookies")
	}
	changed := lanJSON(server, http.MethodPut, "/api/v1/session/password", lanOrigin, `{"current":"old-password-ok","next":"new-password-ok"}`, "192.168.1.50:9", oldCookie)
	if changed.Code != http.StatusNoContent {
		t.Fatalf("change = %d %s", changed.Code, changed.Body.String())
	}
	newCookie := sessionCookie(changed)
	if newCookie == nil {
		t.Fatal("change must reissue cookie")
	}
	probe := func(cookie *http.Cookie) int {
		req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
		req.AddCookie(cookie)
		rec := httptest.NewRecorder()
		server.auth(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })).ServeHTTP(rec, req)
		return rec.Code
	}
	if code := probe(otherCookie); code != http.StatusUnauthorized {
		t.Fatalf("other session = %d", code)
	}
	if code := probe(oldCookie); code != http.StatusUnauthorized {
		t.Fatalf("old session = %d", code)
	}
	if code := probe(newCookie); code != http.StatusNoContent {
		t.Fatalf("new session = %d", code)
	}
}

func TestSessionLogoutClearsCookie(t *testing.T) {
	server := newLANControl(t)
	setup := lanJSON(server, http.MethodPost, "/api/v1/session/setup", lanOrigin, `{"password":"long-enough-password"}`, "192.168.1.50:9")
	cookie := sessionCookie(setup)
	out := lanJSON(server, http.MethodPost, "/api/v1/session/logout", lanOrigin, "", "192.168.1.50:9", cookie)
	if out.Code != http.StatusNoContent {
		t.Fatalf("logout = %d %s", out.Code, out.Body.String())
	}
	cleared := sessionCookie(out)
	if cleared == nil || cleared.MaxAge >= 0 && cleared.Value != "" {
		t.Fatalf("logout cookie = %#v", cleared)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	server.auth(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })).ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("after logout = %d", rec.Code)
	}
}

func TestBearerAndBootstrapBypassGUIPassword(t *testing.T) {
	server := newLANControl(t)
	if rec := lanJSON(server, http.MethodPost, "/api/v1/session/setup", lanOrigin, `{"password":"long-enough-password"}`, "192.168.1.50:9"); rec.Code != http.StatusNoContent {
		t.Fatal(rec.Body.String())
	}
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("Authorization", "Bearer "+server.token)
	rec := httptest.NewRecorder()
	server.auth(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })).ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("bearer = %d %s", rec.Code, rec.Body.String())
	}

	bootstrap := httptest.NewRequest(http.MethodGet, server.BootstrapURL(), nil)
	bootstrap.Host = "192.168.1.20:61767"
	bootRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(bootRec, bootstrap)
	if bootRec.Code != http.StatusFound || sessionCookie(bootRec) == nil {
		t.Fatalf("bootstrap = %d cookies=%v", bootRec.Code, bootRec.Result().Cookies())
	}
	req = httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.AddCookie(sessionCookie(bootRec))
	rec = httptest.NewRecorder()
	server.auth(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })).ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("bootstrap session = %d %s", rec.Code, rec.Body.String())
	}
}
