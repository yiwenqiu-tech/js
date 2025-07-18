package main

import (
	"jieyou-backend/internal/db"
	"jieyou-backend/internal/logic"
)

func main() {
	db.InitDB()
	// 启动Gin路由
	router := logic.SetupRouter()
	router.Run(":8080")
}
