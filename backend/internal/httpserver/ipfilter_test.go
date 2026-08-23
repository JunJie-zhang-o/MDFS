package httpserver

import (
	"net/http/httptest"
	"net/netip"
	"testing"

	"mdfs/internal/config"
)

func TestIPRules(t *testing.T) {
	rules, err := newIPRules(config.Access{Allow: []string{"192.168.0.0/16", "10.0.0.5-10.0.0.10"}, Deny: []string{"192.168.1.7"}, TrustedProxies: []string{"127.0.0.1"}})
	if err != nil {
		t.Fatal(err)
	}
	if !rules.allowed(netip.MustParseAddr("192.168.2.3")) || rules.allowed(netip.MustParseAddr("192.168.1.7")) || !rules.allowed(netip.MustParseAddr("10.0.0.8")) {
		t.Fatal("unexpected allow/deny result")
	}
	request := httptest.NewRequest("GET", "/", nil)
	request.RemoteAddr = "127.0.0.1:9000"
	request.Header.Set("X-Forwarded-For", "192.168.2.4")
	if got := rules.clientIP(request); got.String() != "192.168.2.4" {
		t.Fatalf("client IP = %s", got)
	}
}
