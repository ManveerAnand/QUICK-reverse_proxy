// helpers.go

package utils

import (
	"net/http"
	"time"
)

// ParseDuration is a utility function that parses a duration string and returns the duration in time.Duration format.
func ParseDuration(durationStr string) (time.Duration, error) {
	return time.ParseDuration(durationStr)
}

// HealthCheck performs a simple health check by making a GET request to the specified URL.
func HealthCheck(url string) (bool, error) {
	resp, err := http.Get(url)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK, nil
}

// LogRequest is a utility function to log HTTP requests.
func LogRequest(r *http.Request) {
	// Log the request details (method, URL, headers, etc.)
	// This is a placeholder for actual logging logic.
}