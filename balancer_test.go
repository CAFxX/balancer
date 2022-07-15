package balancer

import (
	"context"
	"fmt"
	"net/http"
	"net/netip"
	"testing"
)

type roundTripperFunc func(req *http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

type resolverFunc func(ctx context.Context, af, host string) ([]netip.Addr, error)

func (f resolverFunc) LookupNetIP(ctx context.Context, af, host string) ([]netip.Addr, error) {
	return f(ctx, af, host)
}

func TestRoundTripperBalance(t *testing.T) {
	addr1, _ := netip.ParseAddr("100.0.0.1")
	addr2, _ := netip.ParseAddr("100.0.0.2")
	m := map[string]int{}

	rt := Wrap(roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		m[req.URL.Hostname()]++
		return nil, nil
	}), resolverFunc(func(ctx context.Context, af, host string) ([]netip.Addr, error) {
		if af != "ip4" {
			t.Fatalf("wrong af: %q", af)
		} else if host != "example.com" {
			t.Fatalf("wrong host: %q", host)
		}
		return []netip.Addr{addr1, addr2}, nil
	}), "ip4")

	for i := 0; i < 1000; i++ {
		req, _ := http.NewRequest(http.MethodGet, "http://example.com:8080/foo", nil)
		rt.RoundTrip(req)
	}

	if m["100.0.0.1"] < 450 || m["100.0.0.1"] > 550 || m["100.0.0.2"] < 450 || m["100.0.0.2"] > 550 {
		t.Fatalf("wrong distribution: %v", m)
	}
}

func TestRoundTripperError(t *testing.T) {
	rt := Wrap(roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		t.Fatal("unreacheable")
		return nil, nil
	}), resolverFunc(func(ctx context.Context, af, host string) ([]netip.Addr, error) {
		return nil, fmt.Errorf("some error")
	}), "ip4")

	req, _ := http.NewRequest(http.MethodGet, "http://foo.example.com", nil)
	_, err := rt.RoundTrip(req)
	if err == nil || err.Error() != `resolving hostname "foo.example.com": some error` {
		t.Fatal(err)
	}
}
