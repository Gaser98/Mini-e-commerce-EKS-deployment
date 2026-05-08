package main

import (
	"database/sql"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"db-design-project/internal/api"
	"db-design-project/internal/db"
)

func main() {
	dbConn, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}

	queries := db.New(dbConn)
	router := gin.Default()
	router.Use(api.PrometheusMiddleware())

	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// CORS middleware
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	api.RegisterRoutes(router, queries)

	// No-cache middleware for static assets
	router.Use(func(c *gin.Context) {
		c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
		c.Header("Pragma", "no-cache")
		c.Header("Expires", "0")
		c.Next()
	})
	router.Static("/static", "./frontend")
	router.GET("/", func(c *gin.Context) {
		c.File("./frontend/index.html")
	})

	router.Run(":8080")
}
