package balancer

import (
	"context"
	"fmt"
	"math/bits"
	"net"
	"net/http"
	"net/netip"

	"github.com/CAFxX/fastrand"
)

type Resolver interface {
	LookupNetIP(ctx context.Context, af, host string) ([]netip.Addr, error)
}

// Wrap returns an http.RoundTripper wrapping the provided http.RoundTripper that
// adds DNS-based load balancing (using the provided Resolver) to requests sent by
// the HTTP client.
//
// The resolver is used to resolve the IP addresses of the hostname in each
// request. The roundtripper does not cache DNS responses, so the resolver is invoked
// for each request (you can use the CachingResolver to add caching to any Resolver).
// net.Resolver and net.DefaultResolver implement the Resolver interface.
//
// The af parameter must be one of "ip4", "ip6", or "ip", and is passed as-is to
// Resolver.LookupNetIP to specify which IP family addresses to use (IPv4, IPv6, or
// both). If your service does not support IPv6 you should set this to "ip4"
// (see net.Resolver for details). Normally the net package automatically attempts
// to use both (see net.Dialer for details), but Wrap modified the request by
// replacing the hostname of the server with one of its IPs (chosen at random, if
// the hostname resolves to multiple IPs), so by the time the request reaches the
// net.Dialer it targets a specific server IP, instead of the server hostname. As
// a result net.Dialer is unable to automatically pick the appropriate IP family.
// For this reason it is extremely important to specify the correct af (address
// family) value.
func Wrap(rt http.RoundTripper, resolver Resolver, af string) http.RoundTripper {
	b := &balancedRoundTripper{rt: rt, resolver: resolver, af: af}
	b.rnd.Seed(fastrand.Seed())
	return b
}

type balancedRoundTripper struct {
	rt       http.RoundTripper
	resolver Resolver
	af       string
	rnd      fastrand.AtomicSplitMix64
}

func (rt *balancedRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	host, hostname := req.URL.Host, req.URL.Hostname()
	ips, err := rt.resolver.LookupNetIP(req.Context(), rt.af, hostname)
	if err != nil || len(ips) == 0 {
		return nil, fmt.Errorf("resolving hostname %q: %w", hostname, err)
	}

	ip := ips[0]
	if len(ips) > 1 {
		// We pick randomly to minimize the chances of thundering herds.
		ip = ips[rt.randIndex(len(ips))]
	}

	// RoundTrippers are not allowed to modify the original request, so we clone the
	// request, modify the clone, and pass the clone to the wrapped RoundTripper.
	req = req.Clone(req.Context())
	if port := req.URL.Port(); port == "" {
		req.URL.Host = ip.String()
	} else {
		req.URL.Host = net.JoinHostPort(ip.String(), port)
	}
	if req.Host == "" && ip.String() != host {
		// Since we replaced the hostname in the URL with an IP, we need to
		// add the hostname in req.Host, or otherwise servers that serve
		// multiple hostnames will not be able to know which hostname the
		// client is referring to.
		req.Host = host
	}

	return rt.rt.RoundTrip(req)
}

func (rt *balancedRoundTripper) randIndex(n int) int {
	hi, _ := bits.Mul64(rt.rnd.Uint64(), uint64(n))
	return int(hi)
}
