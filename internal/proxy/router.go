package proxy

import (
	"fmt"
	"net/http"
	"path"
	"sort"
	"strings"

	"github.com/os-dev/quic-reverse-proxy/internal/config"
)

// Router handles request routing based on configured rules
type Router struct {
	rules          []config.RouteRule
	defaultBackend string
	backends       map[string]*config.BackendConfig
}

// NewRouter creates a new router with the given configuration
func NewRouter(cfg *config.Config) (*Router, error) {
	if len(cfg.Backends) == 0 {
		return nil, fmt.Errorf("no backends configured")
	}

	// Create backend map for quick lookup
	backends := make(map[string]*config.BackendConfig)
	for i := range cfg.Backends {
		backends[cfg.Backends[i].Name] = &cfg.Backends[i]
	}

	// Sort rules by priority (highest first)
	rules := make([]config.RouteRule, len(cfg.Routing.Rules))
	copy(rules, cfg.Routing.Rules)
	sort.Slice(rules, func(i, j int) bool {
		return rules[i].Priority > rules[j].Priority
	})

	// Set default backend
	defaultBackend := cfg.Routing.DefaultBackend
	if defaultBackend == "" && len(cfg.Backends) > 0 {
		defaultBackend = cfg.Backends[0].Name
	}

	return &Router{
		rules:          rules,
		defaultBackend: defaultBackend,
		backends:       backends,
	}, nil
}

// Route finds the appropriate backend for the given request
func (r *Router) Route(req *http.Request) (*config.BackendConfig, error) {
	// Try each rule in priority order
	for _, rule := range r.rules {
		if r.matchRule(req, &rule) {
			backend, ok := r.backends[rule.Backend]
			if !ok {
				return nil, fmt.Errorf("backend not found: %s", rule.Backend)
			}
			return backend, nil
		}
	}

	// Fall back to default backend
	if r.defaultBackend != "" {
		backend, ok := r.backends[r.defaultBackend]
		if ok {
			return backend, nil
		}
	}

	return nil, fmt.Errorf("no matching route found for: %s %s", req.Method, req.URL.Path)
}

// matchRule checks if a request matches a routing rule
func (r *Router) matchRule(req *http.Request, rule *config.RouteRule) bool {
	// Check path pattern
	if rule.Path != "" {
		if !r.matchPath(req.URL.Path, rule.Path) {
			return false
		}
	}

	// Check path prefix
	if rule.PathPrefix != "" {
		if !strings.HasPrefix(req.URL.Path, rule.PathPrefix) {
			return false
		}
	}

	// Check host
	if rule.Host != "" {
		if req.Host != rule.Host {
			return false
		}
	}

	// Check HTTP methods
	if len(rule.Methods) > 0 {
		methodMatch := false
		for _, method := range rule.Methods {
			if strings.EqualFold(req.Method, method) {
				methodMatch = true
				break
			}
		}
		if !methodMatch {
			return false
		}
	}

	// Check headers
	if len(rule.Headers) > 0 {
		for key, value := range rule.Headers {
			if req.Header.Get(key) != value {
				return false
			}
		}
	}

	return true
}

// matchPath checks if a path matches a pattern (supports wildcards)
func (r *Router) matchPath(requestPath, pattern string) bool {
	// Clean paths
	requestPath = path.Clean(requestPath)
	pattern = path.Clean(pattern)

	// Exact match
	if requestPath == pattern {
		return true
	}

	// Wildcard match: /api/* matches /api/users, /api/posts, etc.
	if strings.HasSuffix(pattern, "/*") {
		prefix := strings.TrimSuffix(pattern, "/*")
		if prefix == "" {
			return true // /* matches everything
		}
		return strings.HasPrefix(requestPath, prefix+"/") || requestPath == prefix
	}

	// Single wildcard: /api/*/details matches /api/123/details
	if strings.Contains(pattern, "*") {
		return r.matchWildcard(requestPath, pattern)
	}

	return false
}

// matchWildcard matches paths with wildcards
func (r *Router) matchWildcard(requestPath, pattern string) bool {
	// Split into segments
	pathSegments := strings.Split(strings.Trim(requestPath, "/"), "/")
	patternSegments := strings.Split(strings.Trim(pattern, "/"), "/")

	if len(pathSegments) != len(patternSegments) {
		return false
	}

	for i := range patternSegments {
		if patternSegments[i] == "*" {
			continue
		}
		if pathSegments[i] != patternSegments[i] {
			return false
		}
	}

	return true
}

// GetBackend returns a backend by name
func (r *Router) GetBackend(name string) (*config.BackendConfig, bool) {
	backend, ok := r.backends[name]
	return backend, ok
}

// GetAllBackends returns all configured backends
func (r *Router) GetAllBackends() map[string]*config.BackendConfig {
	return r.backends
}

// ShouldStripPrefix checks if the prefix should be stripped for this request
func (r *Router) ShouldStripPrefix(req *http.Request) (bool, string) {
	for _, rule := range r.rules {
		if r.matchRule(req, &rule) && rule.StripPrefix {
			if rule.PathPrefix != "" {
				return true, rule.PathPrefix
			}
			if rule.Path != "" && strings.HasSuffix(rule.Path, "/*") {
				prefix := strings.TrimSuffix(rule.Path, "/*")
				return true, prefix
			}
		}
	}
	return false, ""
}
