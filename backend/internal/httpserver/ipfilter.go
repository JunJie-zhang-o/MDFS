package httpserver

import (
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"strings"

	"mdfs/internal/config"
)

type ipRange struct {
	prefix      netip.Prefix
	first, last netip.Addr
}

type ipRules struct {
	allow   []ipRange
	deny    []ipRange
	trusted []ipRange
}

func newIPRules(cfg config.Access) (*ipRules, error) {
	rules := &ipRules{}
	var err error
	if rules.allow, err = parseRanges(cfg.Allow); err != nil {
		return nil, fmt.Errorf("access.allow: %w", err)
	}
	if rules.deny, err = parseRanges(cfg.Deny); err != nil {
		return nil, fmt.Errorf("access.deny: %w", err)
	}
	if rules.trusted, err = parseRanges(cfg.TrustedProxies); err != nil {
		return nil, fmt.Errorf("access.trusted_proxies: %w", err)
	}
	return rules, nil
}

func parseRanges(values []string) ([]ipRange, error) {
	result := make([]ipRange, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if prefix, err := netip.ParsePrefix(value); err == nil {
			result = append(result, ipRange{prefix: prefix.Masked()})
			continue
		}
		if first, lastText, ok := strings.Cut(value, "-"); ok {
			start, err1 := netip.ParseAddr(strings.TrimSpace(first))
			end, err2 := netip.ParseAddr(strings.TrimSpace(lastText))
			if err1 != nil || err2 != nil || start.BitLen() != end.BitLen() || end.Less(start) {
				return nil, fmt.Errorf("invalid range %q", value)
			}
			result = append(result, ipRange{first: start.Unmap(), last: end.Unmap()})
			continue
		}
		address, err := netip.ParseAddr(value)
		if err != nil {
			return nil, fmt.Errorf("invalid address %q", value)
		}
		address = address.Unmap()
		result = append(result, ipRange{first: address, last: address})
	}
	return result, nil
}

func (r *ipRules) clientIP(request *http.Request) netip.Addr {
	host, _, err := net.SplitHostPort(request.RemoteAddr)
	if err != nil {
		host = request.RemoteAddr
	}
	remote, _ := netip.ParseAddr(host)
	remote = remote.Unmap()
	if contains(r.trusted, remote) {
		if forwarded := strings.TrimSpace(strings.Split(request.Header.Get("X-Forwarded-For"), ",")[0]); forwarded != "" {
			if address, err := netip.ParseAddr(forwarded); err == nil {
				return address.Unmap()
			}
		}
	}
	return remote
}

func (r *ipRules) allowed(address netip.Addr) bool {
	if contains(r.deny, address) {
		return false
	}
	return len(r.allow) == 0 || contains(r.allow, address)
}

func contains(ranges []ipRange, address netip.Addr) bool {
	if !address.IsValid() {
		return false
	}
	for _, candidate := range ranges {
		if candidate.prefix.IsValid() {
			if candidate.prefix.Contains(address) {
				return true
			}
			continue
		}
		if candidate.first.BitLen() == address.BitLen() && !address.Less(candidate.first) && !candidate.last.Less(address) {
			return true
		}
	}
	return false
}
