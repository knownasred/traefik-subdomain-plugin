package plugindemo_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/traefik/plugindemo"
)

func TestTenant(t *testing.T) {
	cases := []struct{ host, want string }{
		{"alice.app.kepa.ch", "alice"},
		{"team-42.app.kepa.ch:443", "team-42"},
		{"ALICE.APP.KEPA.CH.:443", "alice"},
		{"app.kepa.ch", ""},
		{".app.kepa.ch", ""},
		{"a.b.app.kepa.ch", ""},
		{"alice.app.kepa.ch.evil.test", ""},
		{"elsewhere.test", ""},
		{"[::1]:443", ""},
		{"", ""},
	}
	for _, tc := range cases {
		t.Run(tc.host, func(t *testing.T) {
			called := false
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				assertTenantHeader(t, r.Header, tc.want)
				if r.URL.RequestURI() != "/api/item?q=1" || r.Host != tc.host {
					t.Error("request host or URL changed")
				}
				w.WriteHeader(http.StatusAccepted)
			})
			handler, err := plugindemo.New(context.Background(), next, plugindemo.CreateConfig(), "tenant")
			if err != nil {
				t.Fatal(err)
			}
			req := httptest.NewRequest(http.MethodGet, "http://backend/api/item?q=1", nil)
			req.Host = tc.host
			req.Header["X-Tenant"] = []string{"spoof", "another"}
			req.Header["x-tenant"] = []string{"spoof"}
			req.Header.Set("X-Forwarded-Host", "spoof.app.kepa.ch")
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, req)
			if !called || recorder.Code != http.StatusAccepted {
				t.Fatal("downstream handler not preserved")
			}
		})
	}
}

func assertTenantHeader(t *testing.T, header http.Header, want string) {
	t.Helper()
	if got := header.Get("X-Tenant"); got != want {
		t.Errorf("tenant = %q, want %q", got, want)
	}
	if want == "" && len(header.Values("X-Tenant")) != 0 {
		t.Error("unmatched header should be omitted")
	}
	// Inspect the raw map: Header.Get would hide a noncanonical spoofed key.
	if _, ok := map[string][]string(header)["x-tenant"]; ok {
		t.Error("noncanonical spoofed header survived")
	}
}

func TestCustomConfig(t *testing.T) {
	cfg := plugindemo.CreateConfig()
	cfg.HostRegex = `^(dev|prod)-([^.]+)\.example\.org$`
	cfg.CaptureGroup = 2
	cfg.HeaderName = "X-Account"
	handler, err := plugindemo.New(context.Background(), http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Account") != "acme" {
			t.Errorf("unexpected headers: %v", r.Header)
		}
	}), cfg, "custom")
	if err != nil {
		t.Fatal(err)
	}
	// Configuration is copied at construction and cannot mutate the handler.
	cfg.CaptureGroup = 99
	req := httptest.NewRequest(http.MethodGet, "http://prod-acme.example.org/", nil)
	req.Header = nil
	handler.ServeHTTP(httptest.NewRecorder(), req)
}

func TestInvalidConfig(t *testing.T) {
	cases := []*plugindemo.Config{
		nil,
		{HostRegex: "[", HeaderName: "X-Tenant", CaptureGroup: 1},
		{HostRegex: "", HeaderName: "X-Tenant", CaptureGroup: 1},
		{HostRegex: "example", HeaderName: "X-Tenant", CaptureGroup: 1},
		{HostRegex: "(.*)", HeaderName: "X-Tenant", CaptureGroup: 0},
		{HostRegex: "(.*)", HeaderName: "X-Tenant", CaptureGroup: -1},
		{HostRegex: "(.*)", HeaderName: "X-Tenant", CaptureGroup: 2},
	}
	for _, header := range []string{"", "Bad Header", "Bad:Header", "X\r\nInjected", "Höst", "Host"} {
		cases = append(cases, &plugindemo.Config{HostRegex: "(.*)", HeaderName: header, CaptureGroup: 1})
	}
	for _, cfg := range cases {
		if _, err := plugindemo.New(context.Background(), http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}), cfg, "bad"); err == nil {
			t.Errorf("accepted invalid config: %+v", cfg)
		}
	}
}
