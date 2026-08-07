package main

import (
	"log"
	"net/http"
	"os"
	"silo40/internal/repository"

	"github.com/gin-gonic/gin"
)

func main() {
	// 初始化数据库
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		// 默认本地开发配置
		dsn = "host=localhost user=postgres password=postgres dbname=silo40 port=5432 sslmode=disable TimeZone=Asia/Shanghai"
	}

	db, err := repository.InitDB(dsn)
	if err != nil {
		log.Fatalf("Could not initialize database: %v", err)
	}
	_ = db // TODO: Pass db to services

	r := gin.Default()

	// 基础路由组
	api := r.Group("/api/v1")
	{
		api.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"status":  "ok",
				"message": "Silo40 Strategy Game Backend is running",
			})
		})
	}

	log.Println("Server starting on :8080...")
	if err := r.Run(":8080"); err != nil {
		log.Fatal("Failed to run server: ", err)
	}
}
