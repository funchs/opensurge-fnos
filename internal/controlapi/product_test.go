package controlapi

import (
	"encoding/json"
	"net/http"
	"runtime"
	"testing"
)

func TestRuntimeProductMatchesGOOS(t *testing.T) {
	got := runtimeProduct()
	if got.Name != "OpenSurge" {
		t.Fatalf("name = %q", got.Name)
	}
	want := "mac"
	if runtime.GOOS == "linux" {
		want = "fnos"
	}
	if got.Edition != want {
		t.Fatalf("edition = %q, want %q on GOOS=%s", got.Edition, want, runtime.GOOS)
	}
}

func TestOverviewIncludesRuntimeProduct(t *testing.T) {
	server := newTestServer(t)
	response := performAuthorized(server, http.MethodGet, "/api/v1/overview", nil)
	if response.Code != http.StatusOK {
		t.Fatalf("overview status=%d body=%s", response.Code, response.Body.String())
	}
	var overview Overview
	if err := json.Unmarshal(response.Body.Bytes(), &overview); err != nil {
		t.Fatal(err)
	}
	want := runtimeProduct()
	if overview.Product != want {
		t.Fatalf("product = %#v, want %#v", overview.Product, want)
	}
}
