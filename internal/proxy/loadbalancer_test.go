package proxy

import (
	"fmt"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/os-dev/quic-reverse-proxy/internal/config"
)

func newIPHashLB(t *testing.T, targets ...string) *LoadBalancer {
	t.Helper()
	lb, err := NewLoadBalancer([]config.BackendConfig{{
		Name:         "pool",
		Targets:      targets,
		LoadBalancer: "ip_hash",
	}}, nil)
	if err != nil {
		t.Fatalf("NewLoadBalancer: %v", err)
	}
	return lb
}

func TestIPHashSameClientSameBackend(t *testing.T) {
	lb := newIPHashLB(t, "http://a:80", "http://b:80", "http://c:80")

	// Same IP, different source ports: must always land on the same backend
	var first *Backend
	for port := 1000; port < 1100; port++ {
		r := httptest.NewRequest("GET", "/", nil)
		r.RemoteAddr = fmt.Sprintf("192.0.2.1:%d", port)
		if first == nil {
			first = lb.GetBackendForConfig("pool", r)
		}
		if got := lb.GetBackendForConfig("pool", r); got != first {
			t.Fatalf("port %d: got %s, want %s", port, got.Name, first.Name)
		}
	}
}

func TestIPHashSpreadsClients(t *testing.T) {
	lb := newIPHashLB(t, "http://a:80", "http://b:80", "http://c:80")

	counts := map[string]int{}
	for i := 0; i < 300; i++ {
		r := httptest.NewRequest("GET", "/", nil)
		r.RemoteAddr = fmt.Sprintf("10.0.%d.%d:5555", i/250, i%250)
		counts[lb.GetBackendForConfig("pool", r).Name]++
	}

	if len(counts) != 3 {
		t.Fatalf("expected all 3 backends to get clients, got %v", counts)
	}
	for name, n := range counts {
		if n < 50 {
			t.Errorf("backend %s got only %d of 300 clients: %v", name, n, counts)
		}
	}
}

func TestIPHashSkipsUnhealthyBackend(t *testing.T) {
	lb := newIPHashLB(t, "http://a:80", "http://b:80", "http://c:80")

	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "198.51.100.7:4000"
	original := lb.GetBackendForConfig("pool", r)

	original.SetHealthy(false)
	failover := lb.GetBackendForConfig("pool", r)
	if failover == nil || failover == original {
		t.Fatalf("expected a different healthy backend, got %v", failover)
	}

	// When the original recovers, the client goes back to it
	original.SetHealthy(true)
	if got := lb.GetBackendForConfig("pool", r); got != original {
		t.Fatalf("expected client to return to %s, got %s", original.Name, got.Name)
	}
}

func TestIPHashAllUnhealthy(t *testing.T) {
	lb := newIPHashLB(t, "http://a:80", "http://b:80")
	for _, b := range lb.backends {
		b.SetHealthy(false)
	}
	r := httptest.NewRequest("GET", "/", nil)
	if got := lb.GetBackendForConfig("pool", r); got != nil {
		t.Fatalf("expected nil when all backends are down, got %s", got.Name)
	}
}

func TestRoundRobinTakesTurnsUnderConcurrency(t *testing.T) {
	lb, err := NewLoadBalancer([]config.BackendConfig{
		{Name: "pool", Targets: []string{"http://a:80", "http://b:80"}, LoadBalancer: "round_robin"},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}

	// 100 goroutines x 20 picks; before the fix this wrote to a map under a
	// read lock and could crash with "concurrent map writes"
	var mu sync.Mutex
	counts := map[string]int{}
	var wg sync.WaitGroup
	for g := 0; g < 100; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 20; i++ {
				b := lb.GetBackendForConfig("pool", httptest.NewRequest("GET", "/", nil))
				mu.Lock()
				counts[b.Name]++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	// An atomic counter gives an exact 50/50 split of 2000 picks
	for name, n := range counts {
		if n != 1000 {
			t.Errorf("%s got %d picks, want exactly 1000: %v", name, n, counts)
		}
	}
}

func TestLeastConnectionsPicksIdlest(t *testing.T) {
	lb, err := NewLoadBalancer([]config.BackendConfig{
		{Name: "pool", Targets: []string{"http://a:80", "http://b:80", "http://c:80"}, LoadBalancer: "least_connections"},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	a, b, c := lb.backends[0], lb.backends[1], lb.backends[2]
	a.IncrementConnections()
	a.IncrementConnections()
	c.IncrementConnections()

	got := lb.GetBackendForConfig("pool", httptest.NewRequest("GET", "/", nil))
	if got != b {
		t.Fatalf("got %s, want the idle backend %s", got.Name, b.Name)
	}
	// Selecting must not change the counts; only the handler does that
	if b.GetConnections() != 0 {
		t.Fatalf("selection changed connection count to %d", b.GetConnections())
	}
}

func TestAlgorithmIsPerPool(t *testing.T) {
	lb, err := NewLoadBalancer([]config.BackendConfig{
		{Name: "rr", Targets: []string{"http://a:80", "http://b:80"}, LoadBalancer: "round_robin"},
		{Name: "sticky", Targets: []string{"http://c:80", "http://d:80"}, LoadBalancer: "ip_hash"},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}

	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "203.0.113.9:1234"
	first := lb.GetBackendForConfig("sticky", r)
	for i := 0; i < 10; i++ {
		if got := lb.GetBackendForConfig("sticky", r); got != first {
			t.Fatalf("sticky pool should use ip_hash, but backend changed to %s", got.Name)
		}
	}
}
