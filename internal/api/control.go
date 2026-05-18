package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"time"

	"github.com/sirupsen/logrus"
)

// ControlServer handles backend control operations
type ControlServer struct {
	port string
}

// NewControlServer creates a new control server
func NewControlServer(port string) *ControlServer {
	return &ControlServer{port: port}
}

// Start starts the control API server
func (cs *ControlServer) Start() error {
	// Serve control panel
	http.HandleFunc("/control", cs.handleControlPanel)

	// API endpoints with CORS
	http.HandleFunc("/api/backend/stop", cs.corsMiddleware(cs.handleStopBackend))
	http.HandleFunc("/api/backend/start", cs.corsMiddleware(cs.handleStartBackend))
	http.HandleFunc("/api/backend/restart", cs.corsMiddleware(cs.handleRestartBackend))
	http.HandleFunc("/api/backends/status", cs.corsMiddleware(cs.handleBackendsStatus))
	http.HandleFunc("/api/load/generate", cs.corsMiddleware(cs.handleGenerateLoad))

	logrus.WithField("port", cs.port).Info("Starting control API server")
	return http.ListenAndServe(":"+cs.port, nil)
}

// corsMiddleware adds CORS headers
func (cs *ControlServer) corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}

// handleControlPanel serves the control panel HTML
func (cs *ControlServer) handleControlPanel(w http.ResponseWriter, r *http.Request) {
	paths := []string{"./www/control.html", "/app/www/control.html", "../www/control.html"}
	for _, p := range paths {
		if _, err := http.Dir(".").Open(p); err == nil {
			http.ServeFile(w, r, p)
			return
		}
	}
	logrus.Error("Control panel HTML not found")
	http.Error(w, "Control panel not found", http.StatusNotFound)
}

// handleStopBackend stops a backend container
func (cs *ControlServer) handleStopBackend(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	name := r.URL.Query().Get("name")
	if name == "" {
		http.Error(w, "Backend name is required", http.StatusBadRequest)
		return
	}

	logrus.WithField("backend", name).Info("Stopping backend via API")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", "stop", name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		logrus.WithError(err).WithField("output", string(output)).Error("Failed to stop backend")
		http.Error(w, fmt.Sprintf("Failed to stop backend: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Backend %s stopped", name),
	})
}

// handleStartBackend starts a backend container
func (cs *ControlServer) handleStartBackend(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	name := r.URL.Query().Get("name")
	if name == "" {
		http.Error(w, "Backend name is required", http.StatusBadRequest)
		return
	}

	logrus.WithField("backend", name).Info("Starting backend via API")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", "start", name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		logrus.WithError(err).WithField("output", string(output)).Error("Failed to start backend")
		http.Error(w, fmt.Sprintf("Failed to start backend: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Backend %s started", name),
	})
}

// handleRestartBackend restarts a backend container
func (cs *ControlServer) handleRestartBackend(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	name := r.URL.Query().Get("name")
	if name == "" {
		http.Error(w, "Backend name is required", http.StatusBadRequest)
		return
	}

	logrus.WithField("backend", name).Info("Restarting backend via API")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", "restart", name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		logrus.WithError(err).WithField("output", string(output)).Error("Failed to restart backend")
		http.Error(w, fmt.Sprintf("Failed to restart backend: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Backend %s restarted", name),
	})
}

// handleBackendsStatus returns status of all containers via docker CLI (simple placeholder)
func (cs *ControlServer) handleBackendsStatus(w http.ResponseWriter, r *http.Request) {
	// For now return a simple OK so UI can function; UI already uses /health for backend details
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"backends": []string{"backend1", "backend2"},
	})
}

// handleGenerateLoad generates load by sending requests to the proxy
func (cs *ControlServer) handleGenerateLoad(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	count := 50 // Default request count
	successCount := 0
	errorCount := 0

	logrus.Info("Generating load: 50 requests")

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	// Send requests sequentially to avoid overwhelming the system
	for i := 0; i < count; i++ {
		resp, err := client.Get("http://localhost:80/")
		if err != nil {
			errorCount++
			logrus.WithError(err).Debug("Load generation request failed")
		} else {
			resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				successCount++
			} else {
				errorCount++
			}
		}

		// Small delay between requests
		if i%10 == 0 && i > 0 {
			time.Sleep(50 * time.Millisecond)
		}
	}

	logrus.WithFields(logrus.Fields{
		"success": successCount,
		"failed":  errorCount,
	}).Info("Load generation complete")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":    true,
		"total":      count,
		"successful": successCount,
		"failed":     errorCount,
		"message":    fmt.Sprintf("Load generation complete: %d successful, %d failed", successCount, errorCount),
	})
}
