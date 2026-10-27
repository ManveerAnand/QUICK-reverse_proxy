package proxy

import (
	"crypto/tls"
	"fmt"
	"hash/fnv"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/os-dev/quic-reverse-proxy/internal/config"
	"github.com/os-dev/quic-reverse-proxy/internal/quic"
	"github.com/os-dev/quic-reverse-proxy/internal/telemetry"
	"github.com/os-dev/quic-reverse-proxy/pkg/health"
	"github.com/sirupsen/logrus"
)

// Backend represents a backend server
type Backend struct {
	Name          string
	URL           string
	Protocol      string
	TLSSkipVerify bool
	Weight        int
	healthy       int32 // atomic bool
	checker       *health.Checker
	mu            sync.RWMutex
	connections   int32 // current connection count for least-connections LB
}

// IsHealthy returns true if the backend is healthy
func (b *Backend) IsHealthy() bool {
	return atomic.LoadInt32(&b.healthy) == 1
}

// SetHealthy sets the health status of the backend
func (b *Backend) SetHealthy(healthy bool) {
	value := int32(0)
	if healthy {
		value = 1
	}
	atomic.StoreInt32(&b.healthy, value)
}

// IncrementConnections increments the connection count
func (b *Backend) IncrementConnections() {
	atomic.AddInt32(&b.connections, 1)
}

// DecrementConnections decrements the connection count
func (b *Backend) DecrementConnections() {
	atomic.AddInt32(&b.connections, -1)
}

// GetConnections returns the current connection count
func (b *Backend) GetConnections() int32 {
	return atomic.LoadInt32(&b.connections)
}

// LoadBalancer manages backend selection and health checking
type LoadBalancer struct {
	backends        []*Backend
	backendsByName  map[string][]*Backend // Group backends by config name
	algorithm       string
	algorithms      map[string]string  // Per-backend-config algorithm (load_balancer in YAML)
	roundRobinIndex map[string]*uint32 // Per-backend-config round-robin counter
	mu              sync.RWMutex
	healthCheckers  map[string]*health.Checker
	stopHealthCheck chan struct{}
	metrics         *telemetry.Metrics
}

// NewLoadBalancer creates a new load balancer
func NewLoadBalancer(configs []config.BackendConfig, metrics *telemetry.Metrics) (*LoadBalancer, error) {
	if len(configs) == 0 {
		return nil, fmt.Errorf("no backends configured")
	}

	lb := &LoadBalancer{
		backends:        make([]*Backend, 0, len(configs)),
		backendsByName:  make(map[string][]*Backend),
		algorithms:      make(map[string]string),
		roundRobinIndex: map[string]*uint32{"default": new(uint32)},
		healthCheckers:  make(map[string]*health.Checker),
		stopHealthCheck: make(chan struct{}),
		metrics:         metrics,
	}

	// Create backends
	for _, cfg := range configs {
		configBackends := make([]*Backend, 0, len(cfg.Targets))

		for _, target := range cfg.Targets {
			// Validate URL
			if _, err := url.Parse(target); err != nil {
				return nil, fmt.Errorf("invalid backend URL %s: %w", target, err)
			}

			backend := &Backend{
				Name:          fmt.Sprintf("%s-%s", cfg.Name, target),
				URL:           target,
				Protocol:      cfg.Protocol,
				TLSSkipVerify: cfg.TLSSkipVerify,
				Weight:        cfg.Weight,
			}

			// Initially mark as healthy
			backend.SetHealthy(true)

			// Create health checker if enabled
			if cfg.HealthCheck.Enabled {
				checker := newHealthChecker(cfg, target)
				backend.checker = checker
				lb.healthCheckers[backend.Name] = checker
			}

			lb.backends = append(lb.backends, backend)
			configBackends = append(configBackends, backend)
		}

		// Group backends by config name
		lb.backendsByName[cfg.Name] = configBackends
		lb.algorithms[cfg.Name] = cfg.LoadBalancer
		lb.roundRobinIndex[cfg.Name] = new(uint32)

		// Fallback algorithm for GetBackend, which is not tied to a backend config
		if lb.algorithm == "" {
			lb.algorithm = cfg.LoadBalancer
		}
	}

	logrus.WithFields(logrus.Fields{
		"backends":  len(lb.backends),
		"algorithm": lb.algorithm,
	}).Info("Load balancer initialized")

	return lb, nil
}

// newHealthChecker builds an active health checker that probes the backend
// over the same protocol the proxy uses to forward to it
func newHealthChecker(cfg config.BackendConfig, target string) *health.Checker {
	var transport http.RoundTripper
	switch {
	case cfg.Protocol == "h3":
		transport = quic.NewRoundTripper(cfg.TLSSkipVerify)
	case cfg.TLSSkipVerify:
		t := http.DefaultTransport.(*http.Transport).Clone()
		t.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		transport = t
	}

	return health.NewChecker(target, cfg.HealthCheck.Path, health.Config{
		Interval:           cfg.HealthCheck.Interval,
		Timeout:            cfg.HealthCheck.Timeout,
		HealthyThreshold:   cfg.HealthCheck.HealthyThreshold,
		UnhealthyThreshold: cfg.HealthCheck.UnhealthyThreshold,
		Transport:          transport,
	})
}

// GetBackendForConfig selects a backend for a specific backend config name
func (lb *LoadBalancer) GetBackendForConfig(configName string, r *http.Request) *Backend {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	backends, ok := lb.backendsByName[configName]
	if !ok || len(backends) == 0 {
		return nil
	}

	// ip_hash hashes over the full pool (not just healthy ones) so a client only
	// moves to another backend when its own backend goes down
	if lb.algorithms[configName] == "ip_hash" {
		return lb.ipHash(r, backends)
	}

	healthyBackends := make([]*Backend, 0, len(backends))
	for _, backend := range backends {
		if backend.IsHealthy() {
			healthyBackends = append(healthyBackends, backend)
		}
	}

	if len(healthyBackends) == 0 {
		return nil
	}

	switch lb.algorithms[configName] {
	case "round_robin":
		return lb.roundRobinForConfig(configName, healthyBackends)
	case "least_connections":
		return lb.leastConnections(healthyBackends)
	case "weighted":
		return lb.weighted(healthyBackends)
	default:
		return lb.roundRobinForConfig(configName, healthyBackends)
	}
}

// GetBackend selects a backend using the configured algorithm
func (lb *LoadBalancer) GetBackend(r *http.Request) *Backend {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	if lb.algorithm == "ip_hash" {
		return lb.ipHash(r, lb.backends)
	}

	healthyBackends := lb.getHealthyBackends()
	if len(healthyBackends) == 0 {
		return nil
	}

	switch lb.algorithm {
	case "round_robin":
		return lb.roundRobinForConfig("default", healthyBackends)
	case "least_connections":
		return lb.leastConnections(healthyBackends)
	case "weighted":
		return lb.weighted(healthyBackends)
	default:
		return lb.roundRobinForConfig("default", healthyBackends)
	}
}

// getHealthyBackends returns only healthy backends
func (lb *LoadBalancer) getHealthyBackends() []*Backend {
	var healthy []*Backend
	for _, backend := range lb.backends {
		if backend.IsHealthy() {
			healthy = append(healthy, backend)
		}
	}
	return healthy
}

// roundRobinForConfig implements round-robin load balancing for a specific config
func (lb *LoadBalancer) roundRobinForConfig(configName string, backends []*Backend) *Backend {
	if len(backends) == 0 {
		return nil
	}

	// Callers only hold the read lock, so the map itself is never written here;
	// each pool's counter is a pointer that is advanced atomically.
	counter, ok := lb.roundRobinIndex[configName]
	if !ok {
		return backends[0]
	}

	next := atomic.AddUint32(counter, 1)
	return backends[int((next-1)%uint32(len(backends)))]
}

// roundRobin implements round-robin load balancing (deprecated, use roundRobinForConfig)
func (lb *LoadBalancer) roundRobin(backends []*Backend) *Backend {
	return lb.roundRobinForConfig("default", backends)
}

// leastConnections implements least-connections load balancing
func (lb *LoadBalancer) leastConnections(backends []*Backend) *Backend {
	if len(backends) == 0 {
		return nil
	}

	var selected *Backend
	minConnections := int32(-1)

	for _, backend := range backends {
		connections := backend.GetConnections()
		if minConnections == -1 || connections < minConnections {
			minConnections = connections
			selected = backend
		}
	}

	// The handler increments/decrements the count around the real request
	return selected
}

// ipHash implements IP-hash load balancing: the same client IP always maps to
// the same backend (sticky sessions without cookies). If that backend is
// unhealthy, it walks forward to the next healthy one.
func (lb *LoadBalancer) ipHash(r *http.Request, backends []*Backend) *Backend {
	if len(backends) == 0 {
		return nil
	}

	// RemoteAddr is "ip:port"; the port changes per connection, so hash only the IP.
	// X-Forwarded-For is not used because clients can set it to anything.
	ip := r.RemoteAddr
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		ip = host
	}

	h := fnv.New32a()
	h.Write([]byte(ip))
	start := int(h.Sum32() % uint32(len(backends)))

	for i := 0; i < len(backends); i++ {
		backend := backends[(start+i)%len(backends)]
		if backend.IsHealthy() {
			return backend
		}
	}

	return nil
}

// weighted implements weighted load balancing
func (lb *LoadBalancer) weighted(backends []*Backend) *Backend {
	if len(backends) == 0 {
		return nil
	}

	// Calculate total weight
	totalWeight := 0
	for _, backend := range backends {
		totalWeight += backend.Weight
	}

	if totalWeight == 0 {
		return lb.roundRobin(backends) // Fallback to round-robin
	}

	// Generate random number
	random := rand.Intn(totalWeight)

	// Select backend based on weight
	currentWeight := 0
	for _, backend := range backends {
		currentWeight += backend.Weight
		if random < currentWeight {
			return backend
		}
	}

	return backends[0] // Fallback
}

// StartHealthChecks starts health checking for all backends
func (lb *LoadBalancer) StartHealthChecks() {
	for name, checker := range lb.healthCheckers {
		go func(n string, c *health.Checker) {
			logrus.WithField("backend", n).Info("Starting health checks")

			ticker := time.NewTicker(c.GetInterval())
			defer ticker.Stop()

			for {
				select {
				case <-ticker.C:
					healthy := c.Check()

					// Find the backend and update its health status
					lb.mu.RLock()
					for _, backend := range lb.backends {
						if backend.Name == n {
							wasHealthy := backend.IsHealthy()
							backend.SetHealthy(healthy)

							// Update Prometheus metric
							if lb.metrics != nil {
								lb.metrics.UpdateBackendHealth(n, healthy)
							}

							if wasHealthy != healthy {
								logrus.WithFields(logrus.Fields{
									"backend": n,
									"healthy": healthy,
								}).Info("Backend health status changed")
							}
							break
						}
					}
					lb.mu.RUnlock()

				case <-lb.stopHealthCheck:
					logrus.WithField("backend", n).Info("Stopping health checks")
					return
				}
			}
		}(name, checker)
	}
}

// StopHealthChecks stops health checking
func (lb *LoadBalancer) StopHealthChecks() {
	close(lb.stopHealthCheck)
}

// UpdateBackends updates the backend configuration
func (lb *LoadBalancer) UpdateBackends(configs []config.BackendConfig) error {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	// Stop existing health checks
	lb.StopHealthChecks()

	// Clear existing backends
	lb.backends = lb.backends[:0]
	lb.backendsByName = make(map[string][]*Backend)
	lb.algorithms = make(map[string]string)
	lb.roundRobinIndex = map[string]*uint32{"default": new(uint32)}
	lb.healthCheckers = make(map[string]*health.Checker)
	lb.stopHealthCheck = make(chan struct{})

	// Add new backends (similar to NewLoadBalancer)
	for _, cfg := range configs {
		configBackends := make([]*Backend, 0, len(cfg.Targets))

		for _, target := range cfg.Targets {
			if _, err := url.Parse(target); err != nil {
				return fmt.Errorf("invalid backend URL %s: %w", target, err)
			}

			backend := &Backend{
				Name:          fmt.Sprintf("%s-%s", cfg.Name, target),
				URL:           target,
				Protocol:      cfg.Protocol,
				TLSSkipVerify: cfg.TLSSkipVerify,
				Weight:        cfg.Weight,
			}

			backend.SetHealthy(true)

			if cfg.HealthCheck.Enabled {
				checker := newHealthChecker(cfg, target)
				backend.checker = checker
				lb.healthCheckers[backend.Name] = checker
			}

			lb.backends = append(lb.backends, backend)
			configBackends = append(configBackends, backend)
		}

		lb.backendsByName[cfg.Name] = configBackends
		lb.algorithms[cfg.Name] = cfg.LoadBalancer
		lb.roundRobinIndex[cfg.Name] = new(uint32)
	}

	// Restart health checks
	go lb.StartHealthChecks()

	logrus.WithField("backends", len(lb.backends)).Info("Backends updated")
	return nil
}

// GetBackendStatus returns the status of all backends
func (lb *LoadBalancer) GetBackendStatus() map[string]interface{} {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	status := make(map[string]interface{})
	for _, backend := range lb.backends {
		status[backend.Name] = map[string]interface{}{
			"url":         backend.URL,
			"healthy":     backend.IsHealthy(),
			"weight":      backend.Weight,
			"connections": backend.GetConnections(),
		}
	}

	return status
}
