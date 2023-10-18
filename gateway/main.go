package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

var rdb *redis.Client

// Service represents a registered microservice
type Service struct {
	Name string `json:"name"`
	Host string `json:"host"`
	Port int    `json:"port"`
}

// Maintain a registry of available services in ServiceRegistry
type ServiceRegistry struct {
	services map[string][]Service
	mu       sync.RWMutex
}

// NewServiceRegistry creates a new ServiceRegistry
func NewServiceRegistry() *ServiceRegistry {
	return &ServiceRegistry{
		services: make(map[string][]Service),
	}
}

// RegisterService registers a new microservice with the registry
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

func fetchAllServices(serviceDiscoveryURL string) ([]Service, error) {
	resp, err := http.Get(serviceDiscoveryURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Service Discovery Unavailable")
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var services []Service
	if err := json.Unmarshal(body, &services); err != nil {
		return nil, err
	}

	return services, nil
}

func main() {
	r := gin.Default()
	concurrentLimit := 10
	taskLimit := make(chan struct{}, concurrentLimit)

	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	// Status endpoint for Gateway service
	r.GET("/gateway/status", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "OK"})
	})

	// Status endpoint for Service Discovery
	serviceDiscoveryURL := "http://localhost:8082/services"
	r.GET("/service_discovery/status", func(c *gin.Context) {
		resp, err := http.Get(serviceDiscoveryURL)
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "Service Discovery Unavailable"})
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "Service Discovery Unavailable"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "OK"})
	})

	// Fetch all services from the Service Discovery
	allServices, err := fetchAllServices(serviceDiscoveryURL)
	if err != nil {
		fmt.Println("Error fetching services:", err)
		return
	}

	// Create a ServiceRegistry and register all fetched services
	registry := NewServiceRegistry()
	for _, service := range allServices {
		registry.RegisterService(service.Name, service.Host, service.Port)
	}

	// Not really needed anymore actually
	authenticate := func(c *gin.Context) {
		fmt.Println("Authentication passed")
		c.Next()
	}

	authorizeAccount := func(c *gin.Context) {
		fmt.Println("Authorization for Account passed")
		c.Next()
	}

	authorizeTemplate := func(c *gin.Context) {
		fmt.Println("Authorization for Template passed")
		c.Next()
	}

	// Function to proxy requests to a specific service
	proxyToService := func(serviceName string, registry *ServiceRegistry) gin.HandlerFunc {
		// Initialize a request-specific variable to store the nextServiceIndex
		var nextServiceIndex int

		return func(c *gin.Context) {
			// Check if the response is cached in Redis
			if c.Request.Method == "GET" && c.Request.URL.Path == "/templates/:template_id" ||
				c.Request.Method == "PUT" && c.Request.URL.Path == "/templates/:template_id" ||
				c.Request.Method == "DELETE" && c.Request.URL.Path == "/templates/:template_id" {
				// Create a cache key that includes both the endpoint path and the HTTP method
				cacheKey := fmt.Sprintf("%s:%s", c.Request.Method, c.Request.URL.RequestURI())
				// Check if the response is cached in Redis
				cachedResponse, err := rdb.Get(context.Background(), cacheKey).Result()
				if err == nil {
					c.Data(http.StatusOK, "application/json", []byte(cachedResponse))
					return
				}
			}
			// Try to acquire a slot from the task limit
			select {
			case taskLimit <- struct{}{}:
				defer func() {
					// Release slot when the task is done
					<-taskLimit
				}()

				// Retrieve all available services from Service Directory based on the received serviceName
				availableServices := registry.GetServices(serviceName)

				if len(availableServices) == 0 {
					c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Service Unavailable"})
					return
				}

				// Use the captured nextServiceIndex
				service := availableServices[nextServiceIndex]

				// Update the nextServiceIndex for the next request
				nextServiceIndex = (nextServiceIndex + 1) % len(availableServices)

				serviceURL := fmt.Sprintf("http://%s:%d%s", service.Host, service.Port, c.Request.URL.RequestURI())

				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				// Prepare the proxy request
				method := c.Request.Method
				body := c.Request.Body

				req, err := http.NewRequestWithContext(ctx, method, serviceURL, body)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Error creating request"})
					return
				}

				// Copy headers from the original request to the proxy request
				for key, values := range c.Request.Header {
					for _, value := range values {
						req.Header.Add(key, value)
					}
				}

				// Send the proxy request to the selected service
				client := &http.Client{}
				client.Timeout = 5 * time.Second
				resp, err := client.Do(req)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Error sending request to Service"})
					return
				}
				defer resp.Body.Close()

				// Read the response body
				responseBody, err := io.ReadAll(resp.Body)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Error reading response from Service"})
					return
				}

				// Set the Content-Type header based on the original response
				c.Header("Content-Type", resp.Header.Get("Content-Type"))

				// Send the response from the service to the gateway response
				c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), responseBody)
				// Store the response in Redis for specified endpoints
				if c.Request.Method == "GET" && c.Request.URL.Path == "/templates/:template_id" ||
					c.Request.Method == "PUT" && c.Request.URL.Path == "/templates/:template_id" ||
					c.Request.Method == "DELETE" && c.Request.URL.Path == "/templates/:template_id" {
					cacheKey := fmt.Sprintf("%s:%s", c.Request.Method, c.Request.URL.RequestURI())
					err = rdb.Set(context.Background(), cacheKey, responseBody, 5*time.Minute).Err()
					if err != nil {
						fmt.Println("Error caching data in Redis:", err)
					}
				}
			default:
				// If task limit is reached return a "Service Unavailable" response
				c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Too many concurrent requests"})
			}
		}
	}

	// Routes and endpoint mappings for "account_service"
	r.POST("/accounts", authenticate, authorizeAccount, proxyToService("account_service", registry))                         // Create an account
	r.GET("/accounts/:account_id/balance", authenticate, authorizeAccount, proxyToService("account_service", registry))      // Get an account balance
	r.POST("/accounts/:account_id/deposit", authenticate, authorizeAccount, proxyToService("account_service", registry))     // Deposit funds into an account
	r.POST("/accounts/:account_id/withdraw", authenticate, authorizeAccount, proxyToService("account_service", registry))    // Withdraw funds from an account
	r.GET("/accounts/:account_id/transactions", authenticate, authorizeAccount, proxyToService("account_service", registry)) // Get transactions for an account

	// Routes and endpoint mappings for "template_service"
	r.GET("/templates", authenticate, authorizeTemplate, proxyToService("template_service", registry))                 // Get all templates of an account
	r.POST("/templates", authenticate, authorizeTemplate, proxyToService("template_service", registry))                // Create a template for an account
	r.GET("/templates/:template_id", authenticate, authorizeTemplate, proxyToService("template_service", registry))    // Get a particular template
	r.PUT("/templates/:template_id", authenticate, authorizeTemplate, proxyToService("template_service", registry))    // Update a particular template
	r.DELETE("/templates/:template_id", authenticate, authorizeTemplate, proxyToService("template_service", registry)) // Delete a particular template

	port := 8080
	fmt.Printf("Gateway listening on port %d...\n", port)
	r.Run(fmt.Sprintf(":%d", port))
}
