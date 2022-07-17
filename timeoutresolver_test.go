package balancer

import (
	"context"
	"net/netip"
	"testing"
	"time"
)

func TestTimeoutResolver(t *testing.T) {
	t.Parallel()

	r := &TimeoutResolver{
		Resolver: resolverFunc(func(ctx context.Context, af, host string) ([]netip.Addr, error) {
			if af != "ip" {
				t.Errorf("wrong af: %q", af)
			}
			if host != "host" {
				t.Errorf("wrong host: %q", host)
			}
			<-ctx.Done()
			return nil, ctx.Err()
		}),
		Timeout: 100 * time.Millisecond,
	}
	start := time.Now()
	r.LookupNetIP(context.Background(), "ip", "host")
	if d := time.Since(start).Milliseconds(); d < 100 || d > 150 {
		t.Errorf("wrong timeout: %v", d)
	}
}

func TestTimeoutResolverDisabled(t *testing.T) {
	t.Parallel()

	r := &TimeoutResolver{
		Resolver: resolverFunc(func(ctx context.Context, af, host string) ([]netip.Addr, error) {
			if af != "ip" {
				t.Errorf("wrong af: %q", af)
			}
			if host != "host" {
				t.Errorf("wrong host: %q", host)
			}
			<-ctx.Done()
			return nil, ctx.Err()
		}),
	}
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	r.LookupNetIP(ctx, "ip", "host")
	if d := time.Since(start).Milliseconds(); d < 200 {
		t.Errorf("wrong timeout: %v", d)
	}
}
