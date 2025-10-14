package health

import (
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// Config represents health check configuration
type Config struct {
	Interval           time.Duration
	Timeout            time.Duration
	HealthyThreshold   int
	UnhealthyThreshold int
}

// Checker performs health checks on backend services
type Checker struct {
	baseURL            string
	path               string
	config             Config
	client             *http.Client
	mu                 sync.RWMutex
	consecutiveSuccess int
	consecutiveFailure int
	isHealthy          bool
}

// NewChecker creates a new health checker
func NewChecker(baseURL, path string, config Config) *Checker {
	// Set default values if not provided
	if config.Interval == 0 {
		config.Interval = 30 * time.Second
	}
	if config.Timeout == 0 {
		config.Timeout = 5 * time.Second
	}
	if config.HealthyThreshold == 0 {
		config.HealthyThreshold = 2
	}
	if config.UnhealthyThreshold == 0 {
		config.UnhealthyThreshold = 3
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: config.Timeout,
		// Don't follow redirects for health checks
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	return &Checker{
		baseURL:   baseURL,
		path:      path,
		config:    config,
		client:    client,
		isHealthy: true, // Start as healthy
	}
}

// Check performs a health check and returns the current health status
func (c *Checker) Check() bool {
	success := c.performCheck()

	c.mu.Lock()
	defer c.mu.Unlock()

	if success {
		c.consecutiveSuccess++
		c.consecutiveFailure = 0

		// Mark as healthy if we've reached the threshold
		if !c.isHealthy && c.consecutiveSuccess >= c.config.HealthyThreshold {
			c.isHealthy = true
			logrus.WithField("url", c.getURL()).Info("Backend marked as healthy")
		}
	} else {
		c.consecutiveFailure++
		c.consecutiveSuccess = 0

		// Mark as unhealthy if we've reached the threshold
		if c.isHealthy && c.consecutiveFailure >= c.config.UnhealthyThreshold {
			c.isHealthy = false
			logrus.WithField("url", c.getURL()).Warning("Backend marked as unhealthy")
		}
	}

	return c.isHealthy
}

// performCheck performs the actual HTTP health check
func (c *Checker) performCheck() bool {
	url := c.getURL()

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"url":   url,
			"error": err.Error(),
		}).Debug("Failed to create health check request")
		return false
	}

	// Add health check specific headers
	req.Header.Set("User-Agent", "quic-reverse-proxy-health-checker/1.0")
	req.Header.Set("Cache-Control", "no-cache")

	start := time.Now()
	resp, err := c.client.Do(req)
	duration := time.Since(start)

	if err != nil {
		logrus.WithFields(logrus.Fields{
			"url":      url,
			"error":    err.Error(),
			"duration": duration.String(),
		}).Debug("Health check request failed")
		return false
	}
	defer resp.Body.Close()

	// Consider 200-299 status codes as healthy
	success := resp.StatusCode >= 200 && resp.StatusCode < 300

	logLevel := logrus.DebugLevel
	if !success {
		logLevel = logrus.WarnLevel
	}

	logrus.WithFields(logrus.Fields{
		"url":      url,
		"status":   resp.StatusCode,
		"duration": duration.String(),
		"success":  success,
	}).Log(logLevel, "Health check completed")

	return success
}

// getURL constructs the full health check URL
func (c *Checker) getURL() string {
	baseURL, err := url.Parse(c.baseURL)
	if err != nil {
		return c.baseURL + c.path
	}

	baseURL.Path = c.path
	return baseURL.String()
}

// IsHealthy returns the current health status
func (c *Checker) IsHealthy() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.isHealthy
}

// GetInterval returns the health check interval
func (c *Checker) GetInterval() time.Duration {
	return c.config.Interval
}

// GetStats returns health check statistics
func (c *Checker) GetStats() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return map[string]interface{}{
		"url":                 c.getURL(),
		"healthy":             c.isHealthy,
		"consecutive_success": c.consecutiveSuccess,
		"consecutive_failure": c.consecutiveFailure,
		"config": map[string]interface{}{
			"interval":            c.config.Interval.String(),
			"timeout":             c.config.Timeout.String(),
			"healthy_threshold":   c.config.HealthyThreshold,
			"unhealthy_threshold": c.config.UnhealthyThreshold,
		},
	}
}

// SetHealthy manually sets the health status (for testing or manual intervention)
func (c *Checker) SetHealthy(healthy bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	oldStatus := c.isHealthy
	c.isHealthy = healthy

	if oldStatus != healthy {
		logrus.WithFields(logrus.Fields{
			"url":        c.getURL(),
			"old_status": oldStatus,
			"new_status": healthy,
		}).Info("Health status manually changed")
	}

	// Reset counters when manually setting status
	c.consecutiveSuccess = 0
	c.consecutiveFailure = 0
}

// Reset resets the health checker state
func (c *Checker) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.isHealthy = true
	c.consecutiveSuccess = 0
	c.consecutiveFailure = 0

	logrus.WithField("url", c.getURL()).Info("Health checker reset")
}
