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

// GetServices returns a list of all registered services
func (s *ServiceRegistry) GetServices() []Service {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var allServices []Service
	for _, services := range s.services {
		allServices = append(allServices, services...)
	}
	return allServices
}

func main() {
	registry := NewServiceRegistry()

	// Register replicas for "account_service" and "template_service" in the service directory
	registry.RegisterService("account_service", "accounts_service", 5000)
	registry.RegisterService("template_service", "template_service", 5005)

	// Status endpoint
	http.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		status := map[string]string{"status": "OK"}
		json.NewEncoder(w).Encode(status)
	})

	// Services endpoint
	http.HandleFunc("/services", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		services := registry.GetServices()
		json.NewEncoder(w).Encode(services)
	})

	port := 8082
	fmt.Printf("Service Discovery listening on port %d...\n", port)
	http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
}
