package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

// Service represents a registered microservice.
type Service struct {
	Name string `json:"name"`
	Host string `json:"host"`
	Port int    `json:"port"`
}

// ServiceRegistry maintains a registry of available services.
type ServiceRegistry struct {
	services map[string][]Service
	mu       sync.RWMutex
}

// NewServiceRegistry creates a new ServiceRegistry.
func NewServiceRegistry() *ServiceRegistry {
	return &ServiceRegistry{
		services: make(map[string][]Service), 
	}
}

// RegisterService registers a new microservice with the registry.
func (s *ServiceRegistry) RegisterService(name, host string, port int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.services[name] = append(s.services[name], Service{Name: name, Host: host, Port: port})
}

// GetServices returns a list of all registered services for a given service name.
func (s *ServiceRegistry) GetServices(name string) []Service {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.services[name]
}

func main() {
	registry := NewServiceRegistry()
	// Status endpoint
	http.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		status := map[string]string{"status": "OK"}
		json.NewEncoder(w).Encode(status)
	})

	// Services endpoint
	http.HandleFunc("/services", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Extract the requested service name from the query parameter
		serviceName := r.URL.Query().Get("service")
		if serviceName == "" {
			// If no service name is provided, return all registered services
			services := registry.services
			json.NewEncoder(w).Encode(services)
		} else {
			// Return services for the specified service name
			services := registry.GetServices(serviceName)
			json.NewEncoder(w).Encode(services)
		}
	})

	port := 8082
	fmt.Printf("Service Discovery listening on port %d...\n", port)
	http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
}
