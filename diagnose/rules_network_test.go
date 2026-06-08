package diagnose

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	errorfamily "github.com/larsartmann/go-error-family"
)

func TestNetworkRuleRunLocalhostDNS(t *testing.T) {
	r := &NetworkRule{}
	err := errorfamily.NewTransient("network.connect", "msg").
		WithContext("host", "localhost")

	result, runErr := r.Run(context.Background(), err)
	if runErr != nil {
		t.Fatalf("Run() error: %v", runErr)
	}
	if result.Status != StatusHealthy {
		t.Errorf("Status = %v, want StatusHealthy (localhost should resolve)", result.Status)
	}
	if result.Details["dns_ips"] == "" {
		t.Error("Expected dns_ips to be populated")
	}
}

func TestNetworkRuleRunTCPConnect(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	host, port, splitErr := net.SplitHostPort(srv.Listener.Addr().String())
	if splitErr != nil {
		t.Fatalf("SplitHostPort: %v", splitErr)
	}

	r := &NetworkRule{}
	err := errorfamily.NewTransient("network.connect", "msg").
		WithContext("host", host).
		WithContext("port", port)

	result, runErr := r.Run(context.Background(), err)
	if runErr != nil {
		t.Fatalf("Run() error: %v", runErr)
	}
	if result.Status != StatusHealthy {
		t.Errorf("Status = %v, want StatusHealthy", result.Status)
	}
	if result.Details["tcp_reachable"] != "true" {
		t.Errorf("tcp_reachable = %q, want %q", result.Details["tcp_reachable"], "true")
	}
}

func TestNetworkRuleRunTCPRefused(t *testing.T) {
	r := &NetworkRule{}
	err := errorfamily.NewTransient("network.connect", "msg").
		WithContext("host", "localhost").
		WithContext("port", "1")

	result, runErr := r.Run(context.Background(), err)
	if runErr != nil {
		t.Fatalf("Run() error: %v", runErr)
	}
	if result.Status != StatusFailed {
		t.Errorf("Status = %v, want StatusFailed", result.Status)
	}
	if result.Details["tcp_reachable"] != "false" {
		t.Errorf("tcp_reachable = %q, want %q", result.Details["tcp_reachable"], "false")
	}
}

func TestNetworkRuleRunDNSFailure(t *testing.T) {
	r := &NetworkRule{}
	err := errorfamily.NewTransient("network.connect", "msg").
		WithContext("host", "this-host-definitely-does-not-exist.invalid")

	result, runErr := r.Run(context.Background(), err)
	if runErr != nil {
		t.Fatalf("Run() error: %v", runErr)
	}
	if result.Status != StatusFailed {
		t.Errorf("Status = %v, want StatusFailed", result.Status)
	}
}

func TestNetworkRuleRunWithURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	host, port, splitErr := net.SplitHostPort(srv.Listener.Addr().String())
	if splitErr != nil {
		t.Fatalf("SplitHostPort: %v", splitErr)
	}

	r := &NetworkRule{}
	err := errorfamily.NewTransient("network.connect", "msg").
		WithContext("host", "http://"+net.JoinHostPort(host, port)+"/path")

	result, runErr := r.Run(context.Background(), err)
	if runErr != nil {
		t.Fatalf("Run() error: %v", runErr)
	}
	if result.Status != StatusHealthy {
		t.Errorf("Status = %v, want StatusHealthy", result.Status)
	}
}

func TestNetworkRuleStripHost(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"https://example.com:8080/path", "example.com"},
		{"http://host.local/some/path", "host.local"},
		{"host.local:9090", "host.local"},
		{"host.local", "host.local"},
		{"192.168.1.1:5432", "192.168.1.1"},
	}

	for _, tt := range tests {
		if got := stripHost(tt.input); got != tt.want {
			t.Errorf("stripHost(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
