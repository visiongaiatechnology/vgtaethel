package security

import (
	"context"
	"errors"
	"net"
	"net/netip"
	"net/url"
	"strings"
)

// PublicHTTPSDestination is a canonical HTTPS target whose complete DNS answer
// set was checked before one address was selected for a pinned connection.
type PublicHTTPSDestination struct {
	URL      string
	Hostname string
	Address  netip.Addr
}

var nonPublicNetworkPrefixes = []netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"),
	netip.MustParsePrefix("100.64.0.0/10"),
	netip.MustParsePrefix("192.0.0.0/24"),
	netip.MustParsePrefix("192.0.2.0/24"),
	netip.MustParsePrefix("198.18.0.0/15"),
	netip.MustParsePrefix("198.51.100.0/24"),
	netip.MustParsePrefix("203.0.113.0/24"),
	netip.MustParsePrefix("224.0.0.0/4"),
	netip.MustParsePrefix("240.0.0.0/4"),
	netip.MustParsePrefix("2001:db8::/32"),
}

func isPublicNetworkAddress(address netip.Addr) bool {
	address = address.Unmap()
	if !address.IsValid() || !address.IsGlobalUnicast() || address.IsPrivate() || address.IsLoopback() || address.IsLinkLocalUnicast() || address.IsLinkLocalMulticast() || address.IsMulticast() || address.IsUnspecified() {
		return false
	}
	for _, prefix := range nonPublicNetworkPrefixes {
		if prefix.Contains(address) {
			return false
		}
	}
	return true
}

func isCanonicalDNSHostname(host string) bool {
	if host == "" || len(host) > 253 || strings.HasSuffix(host, ".") {
		return false
	}
	for _, label := range strings.Split(host, ".") {
		if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for _, char := range label {
			if (char < 'a' || char > 'z') && (char < '0' || char > '9') && char != '-' {
				return false
			}
		}
	}
	return true
}

// ResolvePublicHTTPSDestination rejects local, private, special-use and mixed
// DNS answer sets. The caller must pin its connection to Address; resolving the
// hostname a second time would reopen a DNS-rebinding window.
func ResolvePublicHTTPSDestination(ctx context.Context, raw string) (PublicHTTPSDestination, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme != "https" || parsed.Hostname() == "" || parsed.User != nil {
		return PublicHTTPSDestination{}, errors.New("target must be a public HTTPS URL without credentials")
	}
	if parsed.Port() != "" && parsed.Port() != "443" {
		return PublicHTTPSDestination{}, errors.New("target port is not permitted")
	}
	if parsed.Fragment != "" {
		parsed.Fragment = ""
	}
	host := strings.ToLower(parsed.Hostname())
	if host == "localhost" || strings.HasSuffix(host, ".localhost") || strings.HasSuffix(host, ".local") {
		return PublicHTTPSDestination{}, errors.New("target hostname is not permitted")
	}

	var addresses []netip.Addr
	if literal, parseErr := netip.ParseAddr(host); parseErr == nil {
		addresses = []netip.Addr{literal.Unmap()}
	} else {
		if !isCanonicalDNSHostname(host) {
			return PublicHTTPSDestination{}, errors.New("target hostname is invalid")
		}
		resolved, lookupErr := net.DefaultResolver.LookupNetIP(ctx, "ip", host)
		if lookupErr != nil || len(resolved) == 0 {
			return PublicHTTPSDestination{}, errors.New("target hostname resolution failed")
		}
		addresses = resolved
	}

	var selected netip.Addr
	for _, address := range addresses {
		address = address.Unmap()
		if !isPublicNetworkAddress(address) {
			return PublicHTTPSDestination{}, errors.New("target resolved to a blocked address")
		}
		if !selected.IsValid() || (!selected.Is4() && address.Is4()) {
			selected = address
		}
	}
	if !selected.IsValid() {
		return PublicHTTPSDestination{}, errors.New("target has no permitted network address")
	}

	if selected.Is6() && strings.Contains(host, ":") {
		parsed.Host = "[" + host + "]"
	} else {
		parsed.Host = host
	}
	return PublicHTTPSDestination{URL: parsed.String(), Hostname: host, Address: selected}, nil
}

// ChromiumHostResolverRules pins the permitted origin and denies resolution
// for every other host, including redirect and subresource targets.
func ChromiumHostResolverRules(target PublicHTTPSDestination) string {
	host := target.Hostname
	address := target.Address.String()
	if target.Address.Is6() {
		address = "[" + address + "]"
		if strings.Contains(host, ":") {
			host = "[" + host + "]"
		}
	}
	return "MAP " + host + " " + address + ", MAP * ~NOTFOUND"
}
