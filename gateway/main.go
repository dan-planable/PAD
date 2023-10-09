	package main

	import (
		"fmt"
		"net/http"
		"github.com/gin-gonic/gin"
	)

	func main() {
		r := gin.Default()
		// Define routes and endpoint mappings
		r.GET("/accounts/:account_id/balance", authenticate, authorizeAccount, proxyToAccountService)
		r.GET("/accounts/:account_id/transactions", authenticate, authorizeAccount, proxyToAccountService)
		r.POST("/accounts/:account_id/deposit", authenticate, authorizeAccount, proxyToAccountService)
		r.POST("/accounts/:account_id/withdraw", authenticate, authorizeAccount, proxyToAccountService)

		r.GET("/templates", authenticate, authorizeTemplate, proxyToTemplateService)
		r.POST("/templates", authenticate, authorizeTemplate, proxyToTemplateService)
		r.GET("/templates/:template_id", authenticate, authorizeTemplate, proxyToTemplateService)
		r.PUT("/templates/:template_id", authenticate, authorizeTemplate, proxyToTemplateService)
		r.DELETE("/templates/:template_id", authenticate, authorizeTemplate, proxyToTemplateService)

		r.Run(":8080") 
	}

	func authenticate(c *gin.Context) {
		fmt.Println("Authentication passed")
		c.Next()
	}

	func authorizeAccount(c *gin.Context) {
		fmt.Println("Authorization for Account passed")
		c.Next()
	}

	func authorizeTemplate(c *gin.Context) {
		fmt.Println("Authorization for Template passed")
		c.Next()
	}

	// Function to proxy requests to the Account Service
	func proxyToAccountService(c *gin.Context) {
		accountServiceURL := "http://localhost:5000"
		// Prepare the proxy request
		url := accountServiceURL + c.Request.RequestURI
		method := c.Request.Method
		body := c.Request.Body

		req, err := http.NewRequest(method, url, body)
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

		// Send the proxy request to the Account Service
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error sending request to Account Service"})
			return
		}
		defer resp.Body.Close()

		// Copy the response from the Account Service to the gateway response
		c.JSON(resp.StatusCode, gin.H{
			"body": resp.Body,
		})
	}

	// Function to proxy requests to the Template Service
	func proxyToTemplateService(c *gin.Context) {
		templateServiceURL := "http://localhost:5001" 
		// Prepare the proxy request
		url := templateServiceURL + c.Request.RequestURI
		method := c.Request.Method
		body := c.Request.Body

		req, err := http.NewRequest(method, url, body)
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

		// Send the proxy request to the Template Service
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error sending request to Template Service"})
			return
		}
		defer resp.Body.Close()

		// Copy the response from the Template Service to the gateway response
		c.JSON(resp.StatusCode, gin.H{
			"body": resp.Body,
		})
	}
