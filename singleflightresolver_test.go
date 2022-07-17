package balancer

import (
	"context"
	"net/netip"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestSingleflightResolver(t *testing.T) {
	t.Parallel()

	addr1, _ := netip.ParseAddr("100.0.0.1")

	var count uint64

	r := &SingleflightResolver{
		Resolver: resolverFunc(func(ctx context.Context, af, host string) ([]netip.Addr, error) {
			if af != "ip" {
				t.Errorf("wrong af: %q", af)
			}
			if host != "host" {
				t.Errorf("wrong host: %q", host)
			}
			atomic.AddUint64(&count, 1)
			select {
			case <-time.After(200 * time.Millisecond):
				return []netip.Addr{addr1}, nil
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}),
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(3)
	start := time.Now()
	go func() {
		defer wg.Done()
		ctx, _ := context.WithTimeout(ctx, 100*time.Millisecond)
		_, err := r.LookupNetIP(ctx, "ip", "host")
		if err == nil {
			t.Error("nil error, timeout expected")
		}
		if d := time.Since(start).Milliseconds(); d < 100 || d > 150 {
			t.Errorf("unexpected duration %v", d)
		}
	}()
	var res1, res2 []netip.Addr
	go func() {
		defer wg.Done()
		ctx, _ := context.WithTimeout(ctx, 500*time.Millisecond)
		addr, err := r.LookupNetIP(ctx, "ip", "host")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if len(addr) != 1 || addr[0] != addr1 {
			t.Errorf("unexpected addr: %v", addr)
		}
		if d := time.Since(start).Milliseconds(); d < 200 || d > 250 {
			t.Errorf("unexpected duration %v", d)
		}
		res1 = addr
	}()
	go func() {
		defer wg.Done()
		ctx, _ := context.WithTimeout(ctx, 500*time.Millisecond)
		addr, err := r.LookupNetIP(ctx, "ip", "host")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if len(addr) != 1 || addr[0] != addr1 {
			t.Errorf("unexpected addr: %v", addr)
		}
		if d := time.Since(start).Milliseconds(); d < 200 || d > 250 {
			t.Errorf("unexpected duration %v", d)
		}
		res2 = addr
	}()
	wg.Wait()

	if count != 1 {
		t.Errorf("unexpected count %d", count)
	}
	if &res1[0] == &res2[0] {
		t.Error("slice not cloned")
	}
}
