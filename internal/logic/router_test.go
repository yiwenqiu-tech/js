package logic

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// 设置测试环境
func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return SetupRouter()
}

// 测试健康检查接口
func TestPingHandler(t *testing.T) {
	router := setupTestRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/ping", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "pong", response["message"])
}

// 测试获取模板ID接口
func TestGetTemplateIDHandler(t *testing.T) {
	router := setupTestRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/template_id", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.NotEmpty(t, response["template_id"])
}

// 测试签到接口 - 缺少必要参数
func TestSignInHandlerMissingParams(t *testing.T) {
	router := setupTestRouter()
	w := httptest.NewRecorder()

	reqBody := map[string]string{}
	jsonBody, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/api/signin", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, 400, w.Code)
}

// 测试破戒接口 - 缺少必要参数
func TestBreakHandlerMissingParams(t *testing.T) {
	router := setupTestRouter()
	w := httptest.NewRecorder()

	reqBody := map[string]string{}
	jsonBody, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/api/break", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, 400, w.Code)
}

// 测试补卡接口 - 缺少必要参数
func TestRetroactiveSignInHandlerMissingParams(t *testing.T) {
	router := setupTestRouter()
	w := httptest.NewRecorder()

	reqBody := map[string]string{}
	jsonBody, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/api/retroactive", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, 400, w.Code)
}

// 测试订阅授权接口 - 缺少必要参数
func TestSubscriptionAuthHandlerMissingParams(t *testing.T) {
	router := setupTestRouter()
	w := httptest.NewRecorder()

	reqBody := map[string]string{}
	jsonBody, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/api/subscription/auth", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, 400, w.Code)
}

// 测试手动触发提醒检查接口
func TestCheckRemindersHandler(t *testing.T) {
	router := setupTestRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/check_reminders", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "打卡提醒检查已执行", response["message"])
}
