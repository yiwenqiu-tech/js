package logic

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"jieyou-backend/internal/common"
)

// WxAccessTokenResponse 微信access token响应
type WxAccessTokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	ErrCode     int    `json:"errcode"`
	ErrMsg      string `json:"errmsg"`
}

// WxTemplateMessage 微信模板消息
type WxTemplateMessage struct {
	Touser     string                 `json:"touser"`
	TemplateID string                 `json:"template_id"`
	Page       string                 `json:"page"`
	Data       map[string]interface{} `json:"data"`
}

// WxTemplateResponse 微信模板消息响应
type WxTemplateResponse struct {
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
	MsgID   int64  `json:"msgid"`
}

var (
	accessToken     string
	accessTokenTime time.Time
	tokenExpiresIn  int
)

// GetWxAccessToken 获取微信access token
func GetWxAccessToken() (string, error) {
	// 检查token是否过期（提前5分钟刷新）
	if accessToken != "" && time.Now().Before(accessTokenTime.Add(time.Duration(tokenExpiresIn-300)*time.Second)) {
		return accessToken, nil
	}

	url := fmt.Sprintf("https://api.weixin.qq.com/cgi-bin/token?grant_type=client_credential&appid=%s&secret=%s",
		common.WxAPPID, common.WxAPPSecret)

	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("获取access token失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %v", err)
	}

	var tokenResp WxAccessTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("解析响应失败: %v", err)
	}

	if tokenResp.ErrCode != 0 {
		return "", fmt.Errorf("微信API错误: %d - %s", tokenResp.ErrCode, tokenResp.ErrMsg)
	}

	accessToken = tokenResp.AccessToken
	accessTokenTime = time.Now()
	tokenExpiresIn = tokenResp.ExpiresIn

	log.Printf("获取新的access token成功，过期时间: %d秒", tokenExpiresIn)
	return accessToken, nil
}

// SendTemplateMessage 发送模板消息
func SendTemplateMessage(openID, page string, data map[string]interface{}) error {
	token, err := GetWxAccessToken()
	if err != nil {
		return fmt.Errorf("获取access token失败: %v", err)
	}

	url := fmt.Sprintf("https://api.weixin.qq.com/cgi-bin/message/subscribe/send?access_token=%s", token)

	message := WxTemplateMessage{
		Touser:     openID,
		TemplateID: common.WxTemplateID,
		Page:       page,
		Data:       data,
	}

	jsonData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("序列化消息失败: %v", err)
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应失败: %v", err)
	}

	var templateResp WxTemplateResponse
	if err := json.Unmarshal(body, &templateResp); err != nil {
		return fmt.Errorf("解析响应失败: %v", err)
	}

	if templateResp.ErrCode != 0 {
		return fmt.Errorf("发送模板消息失败: %d - %s", templateResp.ErrCode, templateResp.ErrMsg)
	}

	log.Printf("发送模板消息成功，消息ID: %d", templateResp.MsgID)
	return nil
}

// SendSignInReminder 发送打卡提醒
func SendSignInReminder(openID, nickname string) error {
	data := map[string]interface{}{
		"thing1": map[string]string{"value": "打卡提醒"},                                   // 标题
		"thing2": map[string]string{"value": "今日尚未打卡"},                                 // 内容
		"time3":  map[string]string{"value": time.Now().Format("2006-01-02 15:04:05")}, // 时间
		"thing4": map[string]string{"value": "请及时完成今日打卡，以保持进度"},                        // 备注
	}

	return SendTemplateMessage(openID, "pages/index/index", data)
}
