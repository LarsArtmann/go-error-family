package diagnose

import (
	"context"
	"fmt"
	"net"
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
	return networkSpec.matches(err)
}

var networkSpec = ruleSpec{
	ContextKeys:   []string{"host", "port", "url", "endpoint", "address", "remote"},
	CodeContains:  []string{"network", "connect", "dial", "timeout"},
	ContextSubstr: []string{"connection refused", "no such host", "i/o timeout"},
}

func (r *NetworkRule) Run(ctx context.Context, err error) (*DiagnosticResult, error) {
	host := r.resolveHost(err)
	port := r.resolvePort(err)

	result := &DiagnosticResult{
		Details:    map[string]string{"host": host, "port": port},
		Confidence: ConfidenceLikely,
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
		conn, dialErr := net.DialTimeout("tcp", addr, 3*time.Second)
		if dialErr != nil {
			result.Status = StatusFailed
			result.Summary = fmt.Sprintf("Cannot connect to %s: %v", addr, dialErr)
			result.Details["tcp_error"] = dialErr.Error()
			result.Details["tcp_reachable"] = "false"
			result.SuggestedFix = fmt.Sprintf(
				"Check connectivity:\n  nc -zv %s %s\n\nCheck firewall rules and service status.",
				host,
				port,
			)
			return result, nil
		}
		_ = conn.Close()
		result.Details["tcp_reachable"] = "true"
	}

	result.Status = StatusHealthy
	result.Summary = fmt.Sprintf("Network connectivity OK for %s (DNS resolves, TCP connects)", host)
	result.Confidence = ConfidenceNotCause // Network is fine — probably not the root cause

	return result, nil
}

func (r *NetworkRule) resolveHost(err error) string {
	v := resolveContextKey(err, []string{"host", "remote", "endpoint"}, "")
	if v == "" {
		return ""
	}
	v = strings.TrimPrefix(v, "http://")
	v = strings.TrimPrefix(v, "https://")
	if idx := strings.Index(v, ":"); idx > 0 {
		v = v[:idx]
	}
	if idx := strings.Index(v, "/"); idx > 0 {
		v = v[:idx]
	}
	return v
}

func (r *NetworkRule) resolvePort(err error) string {
	return resolveContextKey(err, []string{"port"}, "")
}
