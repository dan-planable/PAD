package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sony/gobreaker"
	"github.com/stathat/consistent"
)

var rdb *redis.Client

// Create a consistent hash ring for cache keys
var cacheRing = consistent.New()

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

// Function to add Redis cache servers to the consistent hash ring
func addCacheServersToRing(redisClients map[string]*redis.Client) {
	for server, client := range redisClients {
		cacheRing.Add(server)
		// Ping each Redis server to check its availability
		if err := client.Ping(context.Background()).Err(); err != nil {
			log.Printf("Error connecting to Redis server %s: %v", server, err)
		}
	}
}

// Function to get the Redis client for a given cache key
func getRedisClientForKey(key string, redisClients map[string]*redis.Client) *redis.Client {
	server, _ := cacheRing.Get(key)
	return redisClients[server]
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

var (
	requestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gateway_requests_total",
			Help: "Total number of requests handled by the gateway.",
		},
		[]string{"status", "method"},
	)

	errorsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gateway_errors_total",
			Help: "Total number of errors encountered by the gateway.",
		},
		[]string{"method"},
	)
)

func main() {
	r := gin.Default()
	concurrentLimit := 10
	taskLimit := make(chan struct{}, concurrentLimit)
	prometheus.MustRegister(requestsTotal, errorsTotal)
	// Create Redis clients for multiple Redis servers
	redisClients := map[string]*redis.Client{
		"redis1": redis.NewClient(&redis.Options{Addr: "redis1:6379", Password: "", DB: 0}),
		"redis2": redis.NewClient(&redis.Options{Addr: "redis2:6379", Password: "", DB: 0}),
		"redis3": redis.NewClient(&redis.Options{Addr: "redis3:6379", Password: "", DB: 0}),
	}

	// Add Redis cache servers to the consistent hash ring
	addCacheServersToRing(redisClients)

	// Status endpoint for Gateway service
	r.GET("/gateway/status", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "OK"})
	})

	// Add a new endpoint for Prometheus metrics
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Status endpoint for Service Discovery
	serviceDiscoveryURL := "http://service_discovery:8082/services"
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

	// Create circuit breakers for account_service and template_service
	accountServiceBreaker := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        "account_service",
		MaxRequests: 1,
		Interval:    4 * 3.5 * time.Second, // reset count each 4 * 3.5 seconds
		Timeout:     5 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= 3
		},
	})

	templateServiceBreaker := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        "template_service",
		MaxRequests: 1,
		Interval:    4 * 3.5 * time.Second, // reset count each 4 * 3.5 seconds
		Timeout:     5 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= 3
		},
	})

	// Function to proxy requests to a specific service with high availability and Prometheus metrics
	proxyToService := func(serviceName string, registry *ServiceRegistry, breaker *gobreaker.CircuitBreaker) gin.HandlerFunc {
		// Initialize a request-specific variable to store the nextServiceIndex and retry count
		var nextServiceIndex int
		var maxRetries = 3

		return func(c *gin.Context) {
			// Increment total requests counter
			requestsTotal.WithLabelValues(strconv.Itoa(http.StatusOK), c.Request.Method).Inc()

			endpoint := c.FullPath()
			method := c.Request.Method
			identifier := ""
			if c.Param("template_id") != "" {
				identifier = c.Param("template_id")
			}

			// Only generate a cache key for specific endpoints
			if (method == "GET" || method == "PUT" || method == "DELETE") && c.Param("template_id") != "" {
				// Combine the method, endpoint, and identifier to create a unique cache key
				cacheKey := fmt.Sprintf("cache:%s:%s:%s", method, endpoint, identifier)

				// Use consistent hashing to determine the Redis server for the key
				redisClient := getRedisClientForKey(cacheKey, redisClients)

				// Try to get cached response from the selected Redis server
				cachedResponse, err := redisClient.Get(context.Background(), cacheKey).Result()
				if err == nil {
					c.Data(http.StatusOK, "application/json", []byte(cachedResponse))
					return
				}
			}

			// Retry loop
			var lastError error
			for retry := 0; retry < maxRetries; retry++ {
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
					// Log which service is being selected for this request
					log.Printf("Selected service: %s, URL: %s", serviceName, serviceURL)
					ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					defer cancel()

					// Prepare the proxy request
					req, err := http.NewRequestWithContext(ctx, method, serviceURL, c.Request.Body)
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
						// Increment total errors counter
						errorsTotal.WithLabelValues(c.Request.Method).Inc()
						c.JSON(http.StatusInternalServerError, gin.H{"error": "Error sending request to Service"})

						lastError = err
						continue // Retry with the next replica
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
					identifier := ""
					if c.Param("template_id") != "" {
						identifier = c.Param("template_id")
					}
					// Only generate a cache key for specific endpoints
					if (method == "GET" || method == "PUT" || method == "DELETE") && c.Param("template_id") != "" {
						// Combine the method, endpoint, and identifier to create a unique cache key
						cacheKey := fmt.Sprintf("cache:%s:%s:%s", method, endpoint, identifier)

						// Use consistent hashing to determine the Redis server for the key
						redisClient := getRedisClientForKey(cacheKey, redisClients)

						// Cache the response on the selected Redis server
						err = redisClient.Set(context.Background(), cacheKey, responseBody, 5*time.Minute).Err()
						if err != nil {
							fmt.Println("Error caching data in Redis:", err)
						}
					}
					// Request succeeded, exit the retry loop
					return
				default:
					// If task limit is reached, return a "Service Unavailable" response
					c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Too many concurrent requests"})
					return
				}
			}
			// If all retries failed, return an error response
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed after %d retries. Last error: %v", maxRetries, lastError)})
		}
	}

	// Routes and endpoint mappings for "account_service"
	r.POST("/accounts", authenticate, authorizeAccount, proxyToService("account_service", registry, accountServiceBreaker))                         // Create an account
	r.GET("/accounts/:account_id/balance", authenticate, authorizeAccount, proxyToService("account_service", registry, accountServiceBreaker))      // Get an account balance
	r.POST("/accounts/:account_id/deposit", authenticate, authorizeAccount, proxyToService("account_service", registry, accountServiceBreaker))     // Deposit funds into an account
	r.POST("/accounts/:account_id/withdraw", authenticate, authorizeAccount, proxyToService("account_service", registry, accountServiceBreaker))    // Withdraw funds from an account
	r.GET("/accounts/:account_id/transactions", authenticate, authorizeAccount, proxyToService("account_service", registry, accountServiceBreaker)) // Get transactions for an account

	// Routes and endpoint mappings for "template_service"
	r.GET("/templates", authenticate, authorizeTemplate, proxyToService("template_service", registry, templateServiceBreaker))                 // Get all templates of an account
	r.POST("/templates", authenticate, authorizeTemplate, proxyToService("template_service", registry, templateServiceBreaker))                // Create a template for an account
	r.GET("/templates/:template_id", authenticate, authorizeTemplate, proxyToService("template_service", registry, templateServiceBreaker))    // Get a particular template
	r.PUT("/templates/:template_id", authenticate, authorizeTemplate, proxyToService("template_service", registry, templateServiceBreaker))    // Update a particular template
	r.DELETE("/templates/:template_id", authenticate, authorizeTemplate, proxyToService("template_service", registry, templateServiceBreaker)) // Delete a particular template

	port := 8080
	fmt.Printf("Gateway listening on port %d...\n", port)
	r.Run(fmt.Sprintf(":%d", port))
}
