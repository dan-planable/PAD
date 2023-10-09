	package main

	import (
		"fmt"
		"context"
		"net/http"
		"time"
		"io"
		"github.com/gin-gonic/gin"
		"encoding/json"
	)

	func main() {
		r := gin.Default()
		// Define routes and endpoint mappings
		r.POST("/accounts", authenticate, authorizeAccount, proxyToAccountService) // Create an account
		r.GET("/accounts/:account_id/balance", authenticate, authorizeAccount, proxyToAccountService) // Get an account balance
		r.GET("/accounts/:account_id/transactions", authenticate, authorizeAccount, proxyToAccountService) // Get account transactions
		r.POST("/accounts/:account_id/deposit", authenticate, authorizeAccount, proxyToAccountService) // Deposit to an account
		r.POST("/accounts/:account_id/withdraw", authenticate, authorizeAccount, proxyToAccountService) // Withdraw from an account

		r.GET("/templates", authenticate, authorizeTemplate, proxyToTemplateService) // Get all templates of an account
		r.POST("/templates", authenticate, authorizeTemplate, proxyToTemplateService) // Create a template for an account
		r.GET("/templates/:template_id", authenticate, authorizeTemplate, proxyToTemplateService) // Get a particular template
		r.PUT("/templates/:template_id", authenticate, authorizeTemplate, proxyToTemplateService) // Update a particular template
		r.DELETE("/templates/:template_id", authenticate, authorizeTemplate, proxyToTemplateService) // Delete a particular template

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
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second) 
		defer cancel()
	
		// Prepare the proxy request
		url := accountServiceURL + c.Request.RequestURI
		method := c.Request.Method
		body := c.Request.Body
	
		req, err := http.NewRequestWithContext(ctx, method, url, body)
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
		client.Timeout = 5 * time.Second 
		resp, err := client.Do(req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error sending request to Account Service"})
			return
		}
		defer resp.Body.Close()
	
		// Read the response body
		responseBody, err := io.ReadAll(resp.Body)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error reading response from Account Service"})
			return
		}
	
		// Marshal the response body into JSON
		var responseJSON interface{}
		err = json.Unmarshal(responseBody, &responseJSON)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error parsing response from Account Service"})
			return
		}
	
		// Copy the response from the Account Service to the gateway response
		c.JSON(resp.StatusCode, responseJSON)
	}

	func proxyToTemplateService(c *gin.Context) {
		templateServiceURL := "http://localhost:5001"
	
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
	
		// Prepare the proxy request
		url := templateServiceURL + c.Request.RequestURI
		method := c.Request.Method
		body := c.Request.Body
	
		req, err := http.NewRequestWithContext(ctx, method, url, body)
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
		client.Timeout = 5 * time.Second 
		resp, err := client.Do(req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error sending request to Template Service"})
			return
		}
		defer resp.Body.Close()
	
		// Read the response body from the Template Service
		responseBody, err := io.ReadAll(resp.Body)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error reading response from Template Service"})
			return
		}

    // Set the Content-Type header based on the original response
    c.Header("Content-Type", resp.Header.Get("Content-Type"))

    // Send the response from the Template Service to the gateway response
    c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), responseBody)
	}

	
	