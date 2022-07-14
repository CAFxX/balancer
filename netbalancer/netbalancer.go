package netbalancer

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"math/bits"
	"math/rand"
	"net"
	"sync/atomic"
	"time"

	"github.com/CAFxX/balancer"
	"github.com/CAFxX/fastrand"
)

// NewNetBalancer returns a Balancer that uses dns lookups from net.Lookup* to reload a set of hosts every updateInterval.
// We can not use TTL from dns because TTL is not exposed by the Go calls.
func New(host string, port int, updateInterval, resolverTimeout time.Duration, resolver resolver) (balancer.Balancer, error) {
	if resolver == nil {
		resolver = net.DefaultResolver
	}

	b := &dnsBalancer{
		lookupAddress:   host,
		port:            port,
		interval:        updateInterval,
		quit:            make(chan struct{}, 1),
		resolverTimeout: resolverTimeout,
		resolver:        resolver,
	}
	b.rnd.Seed(rand.Uint64())

	initialHosts, err := b.lookupTimeout(host, port)
	if err != nil {
		return nil, err
	}
	if len(initialHosts) == 0 {
		return nil, balancer.ErrNoHosts
	}

	b.hosts.Store(initialHosts)

	// start update loop
	go b.loop()

	return b, nil
}

type resolver interface {
	LookupIPAddr(ctx context.Context, host string) ([]net.IPAddr, error)
}

type dnsBalancer struct {
	resolver        resolver
	lookupAddress   string
	port            int
	interval        time.Duration
	quit            chan struct{}
	log             log.Logger
	resolverTimeout time.Duration
	rnd             fastrand.AtomicSplitMix64
	hosts           atomic.Value // []balancer.Host
}

func (b *dnsBalancer) Next() (balancer.Host, error) {
	hosts, _ := b.hosts.Load().([]balancer.Host)

	count := uint64(len(hosts))
	if count == 0 {
		return balancer.Host{}, balancer.ErrNoHosts
	}

	idx := fastmod(b.rnd.Uint64(), count)
	return hosts[idx], nil
}

func (b *dnsBalancer) loop() {
	tick := time.NewTicker(b.interval)
	defer tick.Stop()

	for {
		select {
		case <-tick.C:
			b.update()
		case <-b.quit:
			return
		}
	}
}

func (b *dnsBalancer) update() {
	nextHostList, err := b.lookupTimeout(b.lookupAddress, b.port)
	if err != nil {
		//  TODO: set hostList to empty?
		b.log.Printf("[DnsBalancers] error looking up dns='%v': %v", b.lookupAddress, err)
		return
	}
	if nextHostList != nil {
		prev, _ := b.hosts.Load().([]balancer.Host)
		if !equals(prev, nextHostList) {
			b.log.Printf("[DnsBalancer] hosts changed dns=%v hosts=%v", b.lookupAddress, nextHostList)
			b.hosts.Store(nextHostList)
		}
	}
}

func equalsHost(a balancer.Host, b balancer.Host) bool {
	if a.Port != b.Port {
		return false
	}

	// dont use IP.Equal because it considers ipv4 and ipv6 address to be the same.
	return bytes.Equal(a.Address, b.Address)
}

func equals(a []balancer.Host, b []balancer.Host) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if !hostListContains(b, a[i]) {
			return false
		}
	}

	return true
}

func hostListContains(hosts []balancer.Host, host balancer.Host) bool {
	for i := range hosts {
		if equalsHost(hosts[i], host) {
			return true
		}
	}

	return false
}

func (b *dnsBalancer) lookupTimeout(host string, port int) ([]balancer.Host, error) {
	ctx, cancel := context.WithTimeout(context.Background(), b.resolverTimeout)
	defer cancel()

	return b.lookup(ctx, host, port)
}

func (b *dnsBalancer) lookup(ctx context.Context, host string, port int) ([]balancer.Host, error) {
	ips, err := b.resolver.LookupIPAddr(ctx, host)
	if err != nil {
		return []balancer.Host{}, fmt.Errorf("looking up host %q: %w", host, err)
	}

	hosts := make([]balancer.Host, 0, len(ips))
	for _, ip := range ips {
		entry := balancer.Host{
			Address: ip.IP,
			Port:    port,
		}
		hosts = append(hosts, entry)
	}

	return hosts, nil
}

func (b *dnsBalancer) Close() error {
	// TODO: wait for exit
	b.quit <- struct{}{}

	return nil
}

func fastmod(rnd uint64, mod uint64) uint64 {
	hi, _ := bits.Mul64(rnd, mod)
	return hi
}
