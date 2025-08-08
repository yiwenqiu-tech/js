package main

import (
	"fmt"
	"os"
	"testing"
)

// 设置测试环境变量
func setupTestEnv() {
	os.Setenv("HUNYUAN_TOKEN", "test_token")
	os.Setenv("WX_APPID", "test_appid")
	os.Setenv("WX_APP_SECRET", "test_secret")
	os.Setenv("WX_TEMPLATE_ID", "test_template_id")
	os.Setenv("MYSQL_DSN", "test_mysql_dsn")
}

// 清理测试环境变量
func cleanupTestEnv() {
	os.Unsetenv("HUNYUAN_TOKEN")
	os.Unsetenv("WX_APPID")
	os.Unsetenv("WX_APP_SECRET")
	os.Unsetenv("WX_TEMPLATE_ID")
	os.Unsetenv("MYSQL_DSN")
}

// TestMain 在所有测试开始前运行
func TestMain(m *testing.M) {
	fmt.Printf("test!!!!!!!")
	setupTestEnv()
	code := m.Run()
	cleanupTestEnv()
	os.Exit(code)
}
