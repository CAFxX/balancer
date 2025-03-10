package balancer

import (
	"context"
	"net/netip"
	"sync"
	"sync/atomic"
	"time"
)

// CachingResolver caches responses from the wrapped DNS resolver for the specified
// amount of time.
//
// CachingResolver does not implement a timeout for DNS queries: for that you can
// use a TimeoutResolver. Similarly, it does not implement concurrent request
// deduplication: for that you can use a SingleflightResolver.
// See ExampleAdvanced for the recommended way of composing these additional
// resolvers.
type CachingResolver struct {
	Resolver Resolver      // Wrapped DNS resolver.
	TTL      time.Duration // How long to cache positive results for. 0 disables caching for positive results.
	NegTTL   time.Duration // How long to cache negative results for. 0 disables caching for negative results.

	mu sync.RWMutex
	m  map[key]result

	count uint64
}

type key struct {
	af   string
	host string
}

type result struct {
	ips []netip.Addr
	err error
	exp time.Time
}

var _ Resolver = &CachingResolver{}

func (c *CachingResolver) LookupNetIP(ctx context.Context, af, host string) ([]netip.Addr, error) {
	var mSize int

	if c.TTL > 0 || c.NegTTL > 0 {
		c.mu.RLock()
		r, ok := c.m[key{af, host}]
		mSize = len(c.m)
		c.mu.RUnlock()

		if ok && r.exp.After(time.Now()) {
			c.sampledCleanupAsync(asyncSamples)
			return clone(r.ips), r.err
		}
	}

	exp := time.Now()
	ips, err := c.Resolver.LookupNetIP(ctx, af, host)
	if (err != nil && ctx.Err() != nil) || (err != nil && c.NegTTL == 0) || (err == nil && c.TTL == 0) {
		// If the context was cancelled we don't cache the result.
		// Similarly if the TTL is 0.
		if mSize > 0 {
			c.sampledCleanupAsync(asyncSamples)
		}
		// No need to clone here, as this slice is not from the cache.
		return ips, err
	}

	if err == nil {
		exp = exp.Add(c.TTL)
	} else {
		exp = exp.Add(c.NegTTL)
	}

	c.mu.Lock()

	if c.m == nil {
		c.m = map[key]result{}
	}

	if r, ok := c.m[key{af, host}]; !ok || r.exp.Before(exp) {
		c.m[key{af, host}] = result{ips, err, exp}
	}

	// Whenever we lock the map to add or update an entry, we also check
	// a small number of random entries to see if they are expired. If so
	// we remove them from the map. This is meant to prevent the map from
	// growing unbounded.
	c.sampledCleanupLocked(lockedSamples)

	c.mu.Unlock()

	return clone(ips), err
}

const (
	asyncInterval = 1024
	asyncSamples  = 10
	lockedSamples = 3
)

func (c *CachingResolver) sampledCleanupAsync(samples int) {
	if atomic.AddUint64(&c.count, 1)%asyncInterval == 0 {
		go func() {
			c.mu.Lock()
			defer c.mu.Unlock()
			c.sampledCleanupLocked(samples)
		}()
	}
}

func (c *CachingResolver) sampledCleanupLocked(samples int) {
	now := time.Now()
	for k, r := range c.m {
		if r.exp.Before(now) {
			delete(c.m, k)
		}
		samples--
		if samples <= 0 {
			break
		}
	}
}

func clone(s []netip.Addr) []netip.Addr {
	return append([]netip.Addr(nil), s...)
}
