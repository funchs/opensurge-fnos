package webui

import (
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"
)

func TestApplyEditionBrandRewritesLinuxShell(t *testing.T) {
	source := []byte(`<meta name="opensurge-edition" content="mac" />
<link rel="icon" href="/opensurge-icon.png" />
<title>OpenSurge for Mac</title>
<img src="/opensurge-icon.png" alt="OpenSurge" />`)
	got := string(applyEditionBrand(source, "fnos"))
	if strings.Contains(got, "OpenSurge for Mac") || strings.Contains(got, `content="mac"`) || strings.Contains(got, "/opensurge-icon.png") {
		t.Fatalf("fnOS brand still contains Mac assets:\n%s", got)
	}
	if !strings.Contains(got, "OpenSurge for fnOS") || !strings.Contains(got, `content="fnos"`) || !strings.Contains(got, "/opensurge-fnos-icon.png") {
		t.Fatalf("fnOS brand missing expected assets:\n%s", got)
	}
	if string(applyEditionBrand(source, "mac")) != string(source) {
		t.Fatal("mac edition must leave the embedded index unchanged")
	}
}

func TestHandlerServesUncachedIndexForCurrentEdition(t *testing.T) {
	response := httptest.NewRecorder()
	Handler().ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if response.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("Cache-Control = %q", response.Header().Get("Cache-Control"))
	}
	body := response.Body.String()
	if runtime.GOOS == "linux" {
		if !strings.Contains(body, "OpenSurge for fnOS") {
			t.Fatalf("linux index missing fnOS title:\n%s", body)
		}
		return
	}
	if !strings.Contains(body, "OpenSurge for Mac") {
		t.Fatalf("darwin index missing Mac title:\n%s", body)
	}
}
