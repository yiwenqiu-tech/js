package db

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
		panic("ENV OF MYSQL_DSN IS EMPTY")
	}
	return &Config{
		MySQLDSN: dsn,
	}
}

func (c *Config) Print() {
	fmt.Println("MySQL DSN:", c.MySQLDSN)
}
