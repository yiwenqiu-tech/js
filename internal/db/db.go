package db

import (
	"fmt"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var db *gorm.DB

func GetDB() *gorm.DB {
	return db
}

func InitDB() {
	cfg := LoadConfig()
	cfg.Print()

	var err error
	db, err = gorm.Open(mysql.Open(cfg.MySQLDSN), &gorm.Config{})
	if err != nil {
		panic("failed to connect database: " + err.Error())
	}
	fmt.Println("Connected to MySQL!")

	// 自动迁移表结构
	db.AutoMigrate(&User{}, &SignRecord{}, &ChatRecord{})
}
