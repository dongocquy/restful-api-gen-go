package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	_ "github.com/denisenkom/go-mssqldb"

	"github.com/yourorg/your-api/api"
)

func main() {
	// Database connection
	connStr := os.Getenv("DB_CONN_STR")
	if connStr == "" {
		connStr = "sqlserver://:@localhost:1433?database=&encrypt=disable"
	}

	db, err := sqlx.Connect("sqlserver", connStr)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	log.Println("Connected to SQL Server")

	// Gin router
	r := gin.Default()

	// Register all generated routes
	api.SetupRoutes(r, db)

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on :%s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
