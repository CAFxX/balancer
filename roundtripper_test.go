package balancer

import (
	"context"
	"fmt"
	"net/http"
	"net/netip"
	"net/url"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
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
		if req.Host != "example.com:8080" {
			t.Errorf("wrong hostname %q", req.Host)
		}
		if req.URL.Port() != "8080" {
			t.Errorf("wrong port %q", req.URL.Port())
		}
		if req.URL.Path != "/foo" {
			t.Errorf("wrong path %q", req.URL.Path)
		}
		m[req.URL.Hostname()]++
		return nil, nil
	}), resolverFunc(func(ctx context.Context, af, host string) ([]netip.Addr, error) {
		if af != "ip4" {
			t.Errorf("wrong af: %q", af)
		}
		if host != "example.com" {
			t.Errorf("wrong host: %q", host)
		}
		return []netip.Addr{addr1, addr2}, nil
	}), "ip4")

	for i := 0; i < 1000; i++ {
		req, _ := http.NewRequest(http.MethodGet, "http://example.com:8080/foo", nil)
		rt.RoundTrip(req)
	}

	if m["100.0.0.1"] < 450 || m["100.0.0.1"] > 550 || m["100.0.0.2"] < 450 || m["100.0.0.2"] > 550 {
		t.Errorf("wrong distribution: %v", m)
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
		t.Error(err)
	}
}

func TestRoundTripperWeirdRequest(t *testing.T) {
	addr1, _ := netip.ParseAddr("100.0.0.1")

	rt := Wrap(roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		if req.Host != "bar.example.com" {
			t.Errorf("wrong hostname %q", req.Host)
		}
		if req.URL.Hostname() != "100.0.0.1" {
			t.Errorf("wrong URL hostname %q", req.URL.Hostname())
		}
		if req.URL.Path != "/yadda" {
			t.Errorf("wrong path %q", req.URL.Path)
		}
		return nil, nil
	}), resolverFunc(func(ctx context.Context, af, host string) ([]netip.Addr, error) {
		if af != "ip4" {
			t.Errorf("wrong af: %q", af)
		}
		if host != "bar.example.com" {
			t.Errorf("wrong host: %q", host)
		}
		return []netip.Addr{addr1}, nil
	}), "ip4")

	reqURL, _ := url.Parse("https://bar.example.com/yadda")
	_, err := rt.RoundTrip(&http.Request{
		Method: http.MethodGet,
		URL:    reqURL,
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRoundtripperAndResolvers(t *testing.T) {
	addr1, _ := netip.ParseAddr("100.0.0.1")

	var resolveCount uint64
	var rtCount uint64

	rt := Wrap(roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Hostname() != "100.0.0.1" {
			t.Errorf("wrong URL hostname: %q", req.URL.Hostname())
		}
		if req.Host != "example.com" {
			t.Errorf("wrong hostname: %q", req.Host)
		}
		atomic.AddUint64(&rtCount, 1)
		return nil, nil
	}), &CachingResolver{
		Resolver: &SingleflightResolver{
			Resolver: &TimeoutResolver{
				Resolver: resolverFunc(func(ctx context.Context, af, host string) ([]netip.Addr, error) {
					if af != "ip4" {
						t.Errorf("wrong af: %q", af)
					}
					if host != "example.com" {
						t.Errorf("wrong host: %q", host)
					}
					atomic.AddUint64(&resolveCount, 1)

					time.Sleep(100 * time.Millisecond) // simulate latency
					return []netip.Addr{addr1}, nil
				}),
				Timeout: 2 * time.Second,
			},
		},
		TTL:    1 * time.Second,
		NegTTL: 250 * time.Millisecond,
	}, "ip4")

	start := time.Now()

	var count uint64
	var wg sync.WaitGroup
	for i := 0; i < runtime.GOMAXPROCS(0); i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)
			for {
				c := atomic.LoadUint64(&count)
				if c > 1000000 {
					break
				}
				if atomic.CompareAndSwapUint64(&count, c, c+1) {
					rt.RoundTrip(req)
				}
			}
		}()
	}
	wg.Wait()

	if resolveCount > uint64(time.Since(start).Seconds())+1 {
		t.Errorf("resolveCount: %d", resolveCount)
	}
	if rtCount != count {
		t.Errorf("rtCount != count: %d != %d", rtCount, count)
	}
}
