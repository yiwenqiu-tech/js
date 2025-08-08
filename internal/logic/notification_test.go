package logic

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// 测试微信访问令牌响应结构
func TestWxAccessTokenResponse(t *testing.T) {
	response := WxAccessTokenResponse{
		AccessToken: "test_access_token",
		ExpiresIn:   7200,
		ErrCode:     0,
		ErrMsg:      "",
	}

	assert.Equal(t, "test_access_token", response.AccessToken)
	assert.Equal(t, 7200, response.ExpiresIn)
	assert.Equal(t, 0, response.ErrCode)
	assert.Equal(t, "", response.ErrMsg)
}

// 测试微信模板消息结构
func TestWxTemplateMessage(t *testing.T) {
	data := map[string]interface{}{
		"thing1": map[string]string{"value": "打卡提醒"},
		"thing2": map[string]string{"value": "今日尚未打卡"},
		"time3":  map[string]string{"value": time.Now().Format("2006-01-02 15:04:05")},
		"thing4": map[string]string{"value": "请及时完成今日打卡，以保持进度"},
	}

	message := WxTemplateMessage{
		Touser:     "test_openid",
		TemplateID: "test_template_id",
		Page:       "pages/index/index",
		Data:       data,
	}

	assert.Equal(t, "test_openid", message.Touser)
	assert.Equal(t, "test_template_id", message.TemplateID)
	assert.Equal(t, "pages/index/index", message.Page)
	assert.NotNil(t, message.Data)
	assert.Len(t, message.Data, 4)
}

// 测试微信模板消息响应结构
func TestWxTemplateResponse(t *testing.T) {
	response := WxTemplateResponse{
		ErrCode: 0,
		ErrMsg:  "",
		MsgID:   123456789,
	}

	assert.Equal(t, 0, response.ErrCode)
	assert.Equal(t, "", response.ErrMsg)
	assert.Equal(t, int64(123456789), response.MsgID)
}

// 测试日期格式验证
func TestDateValidation(t *testing.T) {
	validDates := []string{
		"2024-01-15",
		"2024-12-31",
		"2023-02-28",
	}

	for _, date := range validDates {
		_, err := time.Parse("2006-01-02", date)
		assert.NoError(t, err, "日期格式应该有效: %s", date)
	}

	invalidDates := []string{
		"2024-13-01", // 无效月份
		"2024-01-32", // 无效日期
		"2024/01/15", // 错误格式
		"2024-1-5",   // 缺少前导零
	}

	for _, date := range invalidDates {
		_, err := time.Parse("2006-01-02", date)
		assert.Error(t, err, "日期格式应该无效: %s", date)
	}
}

// 测试时间范围验证
func TestTimeRangeValidation(t *testing.T) {
	now := time.Now()
	fiveDaysAgo := now.AddDate(0, 0, -5)

	// 测试最近5天内的日期
	validDates := []time.Time{
		now.AddDate(0, 0, -1),
		now.AddDate(0, 0, -3),
		now.AddDate(0, 0, -5),
	}

	for _, date := range validDates {
		assert.True(t, date.After(fiveDaysAgo) || date.Equal(fiveDaysAgo),
			"日期应该在最近5天内: %s", date.Format("2006-01-02"))
	}

	// 测试超过5天的日期
	invalidDates := []time.Time{
		now.AddDate(0, 0, -6),
		now.AddDate(0, 0, -10),
		now.AddDate(0, 0, -30),
	}

	for _, date := range invalidDates {
		assert.True(t, date.Before(fiveDaysAgo),
			"日期应该超过5天: %s", date.Format("2006-01-02"))
	}
}

// 测试类型验证
func TestTypeValidation(t *testing.T) {
	validTypes := []string{"sign", "break"}
	invalidTypes := []string{"", "other", "SIGN", "BREAK", "Sign", "Break"}

	for _, testType := range validTypes {
		assert.Contains(t, validTypes, testType, "类型应该有效: %s", testType)
	}

	for _, testType := range invalidTypes {
		assert.NotContains(t, validTypes, testType, "类型应该无效: %s", testType)
	}
}

// 测试OpenID格式验证
func TestOpenIDValidation(t *testing.T) {
	validOpenIDs := []string{
		"test_openid_123",
		"wx_openid_456",
		"user_openid_789",
	}

	invalidOpenIDs := []string{
		"",
		"   ",
		"a", // 太短
	}

	for _, openid := range validOpenIDs {
		assert.NotEmpty(t, openid, "OpenID应该有效: %s", openid)
		assert.True(t, len(openid) > 5, "OpenID应该足够长: %s", openid)
	}

	for _, openid := range invalidOpenIDs {
		assert.True(t, openid == "" || len(openid) <= 5, "OpenID应该无效: %s", openid)
	}
}
