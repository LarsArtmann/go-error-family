package diagnose

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"
)

// NetworkRule diagnoses network-related errors.
// Checks: DNS resolution, TCP connectivity, port reachability.
//
// Matches errors with context containing: host, port, url, endpoint, address,
// or Transient family errors, or error codes containing "network", "connect", "dial", "timeout".
type NetworkRule struct{}

func (r *NetworkRule) Name() string { return "network" }

func (r *NetworkRule) Applicable(err error) bool {
	return networkSpec.Matches(err)
}

var networkSpec = RuleSpec{
	ContextKeys:   []ContextKey{KeyHost, KeyPort, KeyURL, KeyEndpoint, KeyAddress, KeyRemote},
	CodeContains:  []string{"network", "connect", "dial", "timeout"},
	ContextSubstr: []string{"connection refused", "no such host", "i/o timeout"},
}

func (r *NetworkRule) Run(ctx context.Context, err error) (*DiagnosticResult, error) {
	host := r.resolveHost(err)
	port := r.resolvePort(err)

	result := &DiagnosticResult{
		Details:    map[string]string{strHost: host, strPort: port},
		Confidence: ConfidenceLikely,
		Context:    ErrorContext(err),
	}

	// Check 1: DNS resolution.
	ips, dnsErr := net.DefaultResolver.LookupHost(ctx, host)
	if dnsErr != nil {
		result.Status = StatusFailed
		result.Summary = fmt.Sprintf("DNS resolution failed for %s: %v", host, dnsErr)
		result.Details["dns_error"] = dnsErr.Error()
		result.SuggestedFix = fmt.Sprintf(
			"Check DNS resolution:\n  dig %s\n  nslookup %s\n\nCheck /etc/hosts or your DNS server.",
			host,
			host,
		)
		return result, nil
	}
	result.Details["dns_ips"] = strings.Join(ips, ", ")

	// Check 2: TCP connectivity.
	if port != "" {
		addr := net.JoinHostPort(host, port)
		dialer := net.Dialer{Timeout: 3 * time.Second}
		conn, dialErr := dialer.DialContext(ctx, "tcp", addr)
		if dialErr != nil {
			result.Status = StatusFailed
			result.Summary = fmt.Sprintf("Cannot connect to %s: %v", addr, dialErr)
			result.Details["tcp_error"] = dialErr.Error()
			result.Details["tcp_reachable"] = strFalse
			result.SuggestedFix = fmt.Sprintf(
				"Check connectivity:\n  nc -zv %s %s\n\nCheck firewall rules and service status.",
				host,
				port,
			)
			return result, nil
		}
		_ = conn.Close()
		result.Details["tcp_reachable"] = strTrue
	}

	result.Status = StatusHealthy
	result.Summary = fmt.Sprintf(
		"Network connectivity OK for %s (DNS resolves, TCP connects)",
		host,
	)
	result.Confidence = ConfidenceNotCause // Network is fine — probably not the root cause

	return result, nil
}

func (r *NetworkRule) resolveHost(err error) string {
	v := ResolveContextKey(err, []string{string(KeyHost), string(KeyRemote), string(KeyEndpoint)}, "")
	if v == "" {
		return ""
	}
	return stripHost(v)
}

// stripHost extracts the hostname from a raw host string or URL.
func stripHost(raw string) string {
	// Try parsing as URL if it has a scheme.
	if strings.Contains(raw, "://") {
		if u, err := url.Parse(raw); err == nil && u.Hostname() != "" {
			return u.Hostname()
		}
	}
	// Strip port manually for bare host:port strings.
	host := raw
	if idx := strings.LastIndex(host, ":"); idx > 0 {
		host = host[:idx]
	}
	// Strip path.
	if idx := strings.Index(host, "/"); idx > 0 {
		host = host[:idx]
	}
	return host
}

func (r *NetworkRule) resolvePort(err error) string {
	return ResolveContextKey(err, []string{string(KeyPort)}, "")
}
