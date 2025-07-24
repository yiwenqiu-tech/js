package db

import (
	"fmt"
	"os"
)

type MysqlConfig struct {
	MySQLDSN string
}

func LoadConfig() *MysqlConfig {
	dsn := os.Getenv("MYSQL_DSN")
	if dsn == "" {
		panic("ENV OF MYSQL_DSN IS EMPTY")
	}
	return &MysqlConfig{
		MySQLDSN: dsn,
	}
}

func (c *MysqlConfig) Print() {
	fmt.Println("MySQL DSN:", c.MySQLDSN)
}
