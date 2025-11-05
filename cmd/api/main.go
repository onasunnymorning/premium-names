package main

import (
	"log"
	"os"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"go.temporal.io/sdk/client"

	"github.com/yourorg/zone-names/internal/api"
	"github.com/yourorg/zone-names/internal/db"
)

func main() {
	// Initialize database
	dbConfig := db.Config{
		Host:     getEnv("DB_HOST", "localhost"),
		Port:     getEnv("DB_PORT", "5432"),
		User:     getEnv("DB_USER", "postgres"),
		Password: getEnv("DB_PASSWORD", "postgres"),
		DBName:   getEnv("DB_NAME", "domain_labels"),
		SSLMode:  getEnv("DB_SSLMODE", "disable"),
	}

	database, err := db.NewDatabase(dbConfig)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	// Initialize Gin
	r := gin.Default()
	r.MaxMultipartMemory = 8 << 20 // 8MB

	// CORS middleware
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	// Serve static files
	r.Static("/static", "./web")
	r.StaticFile("/", "./web/index.html")
	r.StaticFile("/index.html", "./web/index.html")

	// Initialize Temporal client
	temporalClient, err := client.Dial(client.Options{
		HostPort:  getEnv("TEMPORAL_ADDRESS", "localhost:7233"),
		Namespace: getEnv("TEMPORAL_NAMESPACE", "default"),
	})
	if err != nil {
		log.Printf("Warning: Failed to connect to Temporal: %v", err)
		// Continue without Temporal for now
	}
	if temporalClient != nil {
		defer temporalClient.Close()
	}

	// API routes
	apiV1 := r.Group("/api/v1")
	{
		handler := api.NewHandler(database.DB)
		uploadHandler := api.NewUploadHandler(database.DB)

		// Label management routes
		apiV1.GET("/labels", handler.GetLabelsWithPagination)
		apiV1.GET("/labels/:id", handler.GetLabel)
		apiV1.DELETE("/labels/:id", handler.DeleteLabel)
		apiV1.PUT("/labels/:id/tags", handler.UpdateLabelTags)

		// Tag routes
		apiV1.GET("/tags", handler.GetTags)
		apiV1.GET("/tags/stats", handler.GetTagStats)

		// Upload routes
		apiV1.POST("/upload", uploadHandler.UploadFile)

		// Workflow routes (only if Temporal is available)
		if temporalClient != nil {
			workflowHandler := api.NewWorkflowHandler(database.DB, temporalClient)
			apiV1.POST("/workflows/domain-labels", workflowHandler.StartDomainLabelWorkflow)
			apiV1.GET("/workflows/:id/status", workflowHandler.GetWorkflowStatus)
		}
	}

	// Start server
	port := getEnv("PORT", "8080")
	log.Printf("Server starting on :%s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
