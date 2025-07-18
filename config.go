package main

import (
	"fmt"
	"os"
)

type Config struct {
	MySQLDSN string
}

func LoadConfig() *Config {
	dsn := os.Getenv("MYSQL_DSN")
	if dsn == "" {
		dsn = "root:123456@tcp(127.0.0.1:33060)/js?charset=utf8mb4&parseTime=True&loc=Local"
	}
	return &Config{
		MySQLDSN: dsn,
	}
}

func (c *Config) Print() {
	fmt.Println("MySQL DSN:", c.MySQLDSN)
}
