package main

import (
    "context"
    "fmt"
    "io"
    "net/http"
    "sync"
    "time"

    "github.com/gin-gonic/gin"
)

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

// GetAvailableServices returns all available services.
func (s *ServiceRegistry) GetAvailableServices() []Service {
    s.mu.RLock()
    defer s.mu.RUnlock()

    var availableServices []Service
    for _, services := range s.services {
        availableServices = append(availableServices, services...)
    }
    return availableServices
}

func main() {
    r := gin.Default()
    registry := NewServiceRegistry()

    // Register replicas for "account_service" and "template_service" in the service directory
    registry.RegisterService("account_service", "localhost", 5000)
    registry.RegisterService("account_service", "localhost", 5001)
    registry.RegisterService("account_service", "localhost", 5002) 

    registry.RegisterService("template_service", "localhost", 5005)
	registry.RegisterService("template_service", "localhost", 5006) 
	registry.RegisterService("template_service", "localhost", 5007) 

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
    }
}


    // Routes and endpoint mappings for "account_service"
    r.POST("/accounts", authenticate, authorizeAccount, proxyToService("account_service", registry)) // Create an account
    r.GET("/accounts/:account_id/balance", authenticate, authorizeAccount, proxyToService("account_service", registry)) // Get an account balance
	r.POST("/accounts/:account_id/deposit", authenticate, authorizeAccount, proxyToService("account_service", registry)) // Deposit funds into an account
	r.POST("/accounts/:account_id/withdraw", authenticate, authorizeAccount, proxyToService("account_service", registry)) // Withdraw funds from an account
	r.GET("/accounts/:account_id/transactions", authenticate, authorizeAccount, proxyToService("account_service", registry)) // Get transactions for an account

    // Routes and endpoint mappings for "template_service"
    r.GET("/templates", authenticate, authorizeTemplate, proxyToService("template_service", registry)) // Get all templates of an account
	r.POST("/templates", authenticate, authorizeTemplate, proxyToService("template_service", registry)) // Create a template for an account
	r.GET("/templates/:template_id", authenticate, authorizeTemplate, proxyToService("template_service", registry)) // Get a particular template
	r.PUT("/templates/:template_id", authenticate, authorizeTemplate, proxyToService("template_service", registry)) // Update a particular template
	r.DELETE("/templates/:template_id", authenticate, authorizeTemplate, proxyToService("template_service", registry)) // Delete a particular template

    port := 8080
    fmt.Printf("Gateway listening on port %d...\n", port)
    r.Run(fmt.Sprintf(":%d", port))
}

	
	