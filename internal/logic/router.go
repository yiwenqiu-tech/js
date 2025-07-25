package logic

import (
	"context"
	"jieyou-backend/internal/common"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/memory"

	"jieyou-backend/internal/db"

	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"sync"

	"log"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	openai "github.com/sashabaranov/go-openai"
	langopenai "github.com/tmc/langchaingo/llms/openai"
	"gorm.io/gorm"
)

const MaxChatPerDay = 10
const MaxTokenPerMsg = 200

// SetupRouter 路由入口
func SetupRouter() *gin.Engine {
	r := gin.Default()

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "pong"})
	})

	r.POST("/api/signin", SignInHandler)
	r.POST("/api/break", BreakHandler)
	r.GET("/api/calendar", CalendarHandler)
	r.GET("/api/rank/month", MonthRankHandler)
	r.GET("/api/rank/total", TotalRankHandler)
	r.POST("/api/chat", ChatHandler)
	r.GET("/api/chat/history", ChatHistoryHandler)
	r.GET("/api/summary", SummaryHandler)
	r.GET("/api/articles", GetArticlesHandler)
	r.POST("/api/article", CreateArticleHandler)
	r.POST("/api/wxlogin", WxLoginHandler)
	r.POST("/api/user/update_nickname", UpdateNicknameHandler)
	r.GET("/ws/ai", AIWebSocketHandler)

	return r
}

// SignInHandler 签到接口
func SignInHandler(c *gin.Context) {
	var req struct {
		OpenID   string `json:"openid"`
		Nickname string `json:"nickname"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.OpenID == "" {
		c.JSON(400, gin.H{"error": "openid required"})
		return
	}
	user, err := getOrCreateUserByOpenID(req.OpenID, req.Nickname)
	if err != nil {
		c.JSON(500, gin.H{"error": "user error"})
		return
	}
	today := time.Now().Format("2006-01-02")
	var count int64
	db.GetDB().Model(&db.SignRecord{}).Where("user_id = ? AND date = ? AND type = ?", user.ID, today, "sign").Count(&count)
	if count > 0 {
		c.JSON(400, gin.H{"error": "already signed in today"})
		return
	}
	// 新增：当天已破戒则禁止守戒签到
	var breakCount int64
	db.GetDB().Model(&db.SignRecord{}).Where("user_id = ? AND date = ? AND type = ?", user.ID, today, "break").Count(&breakCount)
	if breakCount > 0 {
		c.JSON(400, gin.H{"error": "今日已破戒"})
		return
	}
	record := db.SignRecord{UserID: user.ID, Date: today, Type: "sign"}
	if err := db.GetDB().Create(&record).Error; err != nil {
		c.JSON(500, gin.H{"error": "db error"})
		return
	}
	c.JSON(200, gin.H{"message": "sign in success"})
}

// BreakHandler 破戒
func BreakHandler(c *gin.Context) {
	var req struct {
		OpenID   string `json:"openid"`
		Nickname string `json:"nickname"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.OpenID == "" {
		c.JSON(400, gin.H{"error": "openid required"})
		return
	}
	user, err := getOrCreateUserByOpenID(req.OpenID, req.Nickname)
	if err != nil {
		c.JSON(500, gin.H{"error": "user error"})
		return
	}
	today := time.Now().Format("2006-01-02")
	var count int64
	db.GetDB().Model(&db.SignRecord{}).Where("user_id = ? AND date = ? AND type = ?", user.ID, today, "break").Count(&count)
	if count > 0 {
		c.JSON(400, gin.H{"error": "already broke today"})
		return
	}
	// 删除当天的sign记录（如果有）
	db.GetDB().Where("user_id = ? AND date = ? AND type = ?", user.ID, today, "sign").Delete(&db.SignRecord{})
	record := db.SignRecord{UserID: user.ID, Date: today, Type: "break"}
	if err := db.GetDB().Create(&record).Error; err != nil {
		c.JSON(500, gin.H{"error": "db error"})
		return
	}
	c.JSON(200, gin.H{"message": "break success"})
}

// CalendarHandler 日历
func CalendarHandler(c *gin.Context) {
	openid := c.Query("openid")
	if openid == "" {
		c.JSON(400, gin.H{"error": "openid required"})
		return
	}
	user := db.User{}
	err := db.GetDB().Where("open_id = ?", openid).First(&user).Error
	if err != nil {
		c.JSON(404, gin.H{"error": "user not found"})
		return
	}
	// 拉取所有记录
	var records []db.SignRecord
	db.GetDB().Where("user_id = ?", user.ID).Order("date asc").Find(&records)

	// 统计数据
	totalSign := 0
	totalBreak := 0
	currentStreak := 0
	lastDate := ""
	streak := 0
	// 先统计累计
	for _, r := range records {
		if r.Type == "sign" {
			totalSign++
		} else if r.Type == "break" {
			totalBreak++
		}
	}
	// 计算当前连续打卡
	for _, r := range records {
		if r.Type == "break" {
			streak = 0
			lastDate = r.Date
			continue
		}
		if r.Type == "sign" {
			if lastDate == "" || nextDay(lastDate) == r.Date {
				streak++
			} else {
				streak = 1
			}
			lastDate = r.Date
		}
	}
	currentStreak = streak

	// 日历展示，优先展示“破”logo
	calendar := map[string]string{} // date: "sign"/"break"
	for _, r := range records {
		if r.Type == "break" {
			calendar[r.Date] = "break"
		} else if r.Type == "sign" {
			if calendar[r.Date] != "break" {
				calendar[r.Date] = "sign"
			}
		}
	}

	c.JSON(200, gin.H{
		"records":        records,
		"calendar":       calendar,
		"total_sign":     totalSign,
		"total_break":    totalBreak,
		"current_streak": currentStreak,
	})
}

// 计算 next day
func nextDay(date string) string {
	t, _ := time.Parse("2006-01-02", date)
	return t.AddDate(0, 0, 1).Format("2006-01-02")
}

// MonthRankHandler 月排行榜
func MonthRankHandler(c *gin.Context) {
	openid := c.Query("open_id")
	type Result struct {
		Nickname string
		Streak   int64
		UserID   uint
		IsSelf   bool
		Rank     int
	}
	var results []Result
	// SQL聚合：查每个用户最后一次break后连续sign天数
	db.GetDB().Raw(`
	SELECT u.id as user_id, u.nickname,
	  COUNT(s.id) as streak
	FROM users u
	LEFT JOIN (
	  SELECT user_id, MAX(CASE WHEN type = 'break' THEN date END) as last_break
	  FROM sign_records
	  GROUP BY user_id
	) b ON u.id = b.user_id
	LEFT JOIN sign_records s
	  ON s.user_id = u.id
	  AND s.type = 'sign'
	  AND (b.last_break IS NULL OR s.date > b.last_break)
	GROUP BY u.id, u.nickname
	ORDER BY streak DESC, u.id ASC
	LIMIT 10
	`).Scan(&results)
	// 排名处理
	for i := range results {
		results[i].Rank = i + 1
	}
	// 标记自己
	if openid != "" {
		var user db.User
		err := db.GetDB().Where("open_id = ?", openid).First(&user).Error
		if err == nil {
			for i := range results {
				if results[i].UserID == user.ID {
					results[i].IsSelf = true
					break
				}
			}
		}
	}
	c.JSON(200, gin.H{"rank": results})
}

// 总排行榜
func TotalRankHandler(c *gin.Context) {
	openid := c.Query("open_id")
	type Result struct {
		Nickname string
		Streak   int64
		UserID   uint
		IsSelf   bool
		Rank     int
	}
	var results []Result
	db.GetDB().Raw(`
	SELECT u.id as user_id, u.nickname,
	  COUNT(s.id) as streak
	FROM users u
	LEFT JOIN (
	  SELECT user_id, MAX(CASE WHEN type = 'break' THEN date END) as last_break
	  FROM sign_records
	  GROUP BY user_id
	) b ON u.id = b.user_id
	LEFT JOIN sign_records s
	  ON s.user_id = u.id
	  AND s.type = 'sign'
	  AND (b.last_break IS NULL OR s.date > b.last_break)
	GROUP BY u.id, u.nickname
	ORDER BY streak DESC, u.id ASC
	LIMIT 10
	`).Scan(&results)
	for i := range results {
		results[i].Rank = i + 1
	}
	if openid != "" {
		var user db.User
		err := db.GetDB().Where("open_id = ?", openid).First(&user).Error
		if err == nil {
			for i := range results {
				if results[i].UserID == user.ID {
					results[i].IsSelf = true
					break
				}
			}
		}
	}
	c.JSON(200, gin.H{"rank": results})
}

// ChatHandler AI 聊天接口
func ChatHandler(c *gin.Context) {
	var req struct {
		OpenID   string `json:"openid"`
		Nickname string `json:"nickname"`
		Content  string `json:"content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.OpenID == "" || req.Content == "" {
		c.JSON(400, gin.H{"error": "openid and content required"})
		return
	}
	user, err := getOrCreateUserByOpenID(req.OpenID, req.Nickname)
	if err != nil {
		c.JSON(500, gin.H{"error": "user error"})
		return
	}
	today := time.Now().Format("2006-01-02")
	var count int64
	db.GetDB().Model(&db.ChatRecord{}).Where("user_id = ? AND is_user = 1 AND DATE(created_at) = ?", user.ID, today).Count(&count)
	if count >= MaxChatPerDay {
		c.JSON(400, gin.H{"error": "今日已达上限"})
		return
	}
	if utf8.RuneCountInString(req.Content) > MaxTokenPerMsg {
		c.JSON(400, gin.H{"error": "消息过长"})
		return
	}
	if strings.Contains(strings.ToLower(req.Content), "openai") || strings.Contains(req.Content, "system") {
		c.JSON(400, gin.H{"error": "消息包含敏感内容"})
		return
	}
	db.GetDB().Create(&db.ChatRecord{UserID: user.ID, Content: req.Content, IsUser: true})
	var history []db.ChatRecord
	db.GetDB().Where("user_id = ?", user.ID).Order("created_at asc").Find(&history)

	ctx := context.Background()
	// TODO 考虑压缩上下文！
	chatMemory := memory.NewConversationWindowBuffer(10)
	chatMemory.ChatHistory.AddUserMessage(ctx, common.RolePrompt)
	for _, h := range history {
		if h.IsUser {
			chatMemory.ChatHistory.AddUserMessage(ctx, h.Content)
		} else {
			chatMemory.ChatHistory.AddAIMessage(ctx, h.Content)
		}
	}
	llm, _ := langopenai.New(
		langopenai.WithToken(common.HunyuanToken),
		langopenai.WithModel(common.HunyuanModel),
		langopenai.WithBaseURL(common.HunyuanBaseUrl))
	chain := chains.NewConversation(llm, chatMemory)
	resp, err := chains.Run(ctx, chain, req.Content, chains.WithMaxTokens(200))
	if err != nil {
		c.JSON(500, gin.H{"error": "AI error"})
		return
	}
	db.GetDB().Create(&db.ChatRecord{UserID: user.ID, Content: resp, IsUser: false})
	c.JSON(200, gin.H{"reply": resp})
}

// ChatHistoryHandler 聊天历史接口
func ChatHistoryHandler(c *gin.Context) {
	openid := c.Query("openid")
	if openid == "" {
		c.JSON(400, gin.H{"error": "openid required"})
		return
	}
	user := db.User{}
	err := db.GetDB().Where("open_id = ?", openid).First(&user).Error
	if err != nil {
		c.JSON(404, gin.H{"error": "user not found"})
		return
	}
	var records []db.ChatRecord
	db.GetDB().Where("user_id = ?", user.ID).Order("created_at asc").Find(&records)
	fmt.Println(records)
	c.JSON(200, gin.H{"records": records})
}

// SummaryHandler 统计汇总接口
func SummaryHandler(c *gin.Context) {
	var totalSign int64
	var totalBreak int64
	var userCount int64
	db.GetDB().Model(&db.SignRecord{}).Where("type = ?", "sign").Count(&totalSign)
	db.GetDB().Model(&db.SignRecord{}).Where("type = ?", "break").Count(&totalBreak)
	db.GetDB().Model(&db.User{}).Count(&userCount)
	c.JSON(200, gin.H{
		"total_sign":  totalSign,
		"total_break": totalBreak,
		"user_count":  userCount,
	})
}

// GetArticlesHandler 拉取文章列表
func GetArticlesHandler(c *gin.Context) {
	var articles []db.Article
	db.GetDB().Order("created_at desc").Find(&articles)
	c.JSON(200, gin.H{"articles": articles})
}

// CreateArticleHandler 创建文章
func CreateArticleHandler(c *gin.Context) {
	var req struct {
		Title string `json:"title"`
		Desc  string `json:"desc"`
		Img   string `json:"img"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Title == "" {
		c.JSON(400, gin.H{"error": "title required"})
		return
	}
	article := db.Article{
		Title:     req.Title,
		Desc:      req.Desc,
		Img:       req.Img,
		CreatedAt: time.Now(),
		ReadCount: 0,
	}
	if err := db.GetDB().Create(&article).Error; err != nil {
		c.JSON(500, gin.H{"error": "db error"})
		return
	}
	c.JSON(200, gin.H{"id": article.ID})
}

// WxLoginHandler 微信登录接口
func WxLoginHandler(c *gin.Context) {
	type Req struct {
		Code     string `json:"code"`
		Nickname string `json:"nickname"`
	}
	var req Req
	if err := c.ShouldBindJSON(&req); err != nil || req.Code == "" {
		c.JSON(400, gin.H{"error": "code required"})
		return
	}
	appid := common.WxAPPID
	secret := common.WxAPPSecret
	url := fmt.Sprintf("https://api.weixin.qq.com/sns/jscode2session?appid=%s&secret=%s&js_code=%s&grant_type=authorization_code", appid, secret, req.Code)
	resp, err := http.Get(url)
	if err != nil {
		c.JSON(500, gin.H{"error": "wx api error", "detail": err.Error()})
		return
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	var wxResp struct {
		OpenID string `json:"openid"`
		ErrMsg string `json:"errmsg"`
	}
	json.Unmarshal(body, &wxResp)
	if wxResp.OpenID == "" {
		c.JSON(400, gin.H{"error": "get openid failed", "detail": string(body)})
		return
	}
	var user db.User
	err = db.GetDB().Where("open_id = ?", wxResp.OpenID).First(&user).Error
	if err == gorm.ErrRecordNotFound {
		user = db.User{OpenID: wxResp.OpenID, Nickname: req.Nickname}
		db.GetDB().Create(&user)
		if user.Nickname == "" {
			user.Nickname = fmt.Sprintf("戒友%d", user.ID)
			db.GetDB().Model(&user).Update("nickname", user.Nickname)
		}
	} else if err != nil {
		c.JSON(500, gin.H{"error": "db error", "detail": err.Error()})
		return
	}
	c.JSON(200, gin.H{"openid": wxResp.OpenID, "nickname": user.Nickname})
}

// 修改昵称接口
func UpdateNicknameHandler(c *gin.Context) {
	type Req struct {
		OpenID   string `json:"openid"`
		Nickname string `json:"nickname"`
	}
	var req Req
	if err := c.ShouldBindJSON(&req); err != nil || req.OpenID == "" || req.Nickname == "" {
		c.JSON(400, gin.H{"error": "openid and nickname required"})
		return
	}
	// 检查昵称是否已被占用
	var count int64
	db.GetDB().Model(&db.User{}).Where("nickname = ? AND open_id != ?", req.Nickname, req.OpenID).Count(&count)
	if count > 0 {
		c.JSON(400, gin.H{"error": "昵称已被占用，请"})
		return
	}
	var user db.User
	err := db.GetDB().Where("open_id = ?", req.OpenID).First(&user).Error
	if err != nil {
		c.JSON(404, gin.H{"error": "user not found"})
		return
	}
	user.Nickname = req.Nickname
	db.GetDB().Model(&user).Update("nickname", req.Nickname)
	c.JSON(200, gin.H{"success": true, "nickname": req.Nickname})
}

// 通过 openid 获取或创建用户
func getOrCreateUserByOpenID(openid, nickname string) (*db.User, error) {
	var user db.User
	err := db.GetDB().Where("open_id = ?", openid).First(&user).Error
	if err == nil {
		return &user, nil
	}
	if err == gorm.ErrRecordNotFound {
		user = db.User{OpenID: openid, Nickname: nickname}
		err = db.GetDB().Create(&user).Error
		if err == nil {
			return &user, nil
		}
	}
	return nil, err
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// AI流式回复会话（仅适合单实例开发环境）
type StreamSession struct {
	History []rune        // 已发送内容
	Done    chan struct{} // 结束信号
}

var aiStreamSessions = make(map[string]*StreamSession) // key: userID+msgID
var aiStreamSessionsLock sync.Mutex

func AIWebSocketHandler(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	// 读取前端发来的 openid、content、msg_id、received_len
	_, msg, err := conn.ReadMessage()
	if err != nil {
		return
	}
	type Req struct {
		OpenID      string `json:"openid"`
		Content     string `json:"content"`
		MsgID       string `json:"msg_id"`
		ReceivedLen int    `json:"received_len"`
	}
	var req Req
	json.Unmarshal(msg, &req)

	// 查找用户
	var user db.User
	err = db.GetDB().Where("open_id = ?", req.OpenID).First(&user).Error
	if err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte("用户不存在"))
		return
	}

	cacheKey := fmt.Sprintf("%d_%s", user.ID, req.MsgID)

	// 先查数据库（已完成的AI回复）
	var aiRecord db.ChatRecord
	if db.GetDB().Where("user_id = ? AND msg_id = ? AND is_user = 0", user.ID, req.MsgID).First(&aiRecord).Error == nil {
		aiRunes := []rune(aiRecord.Content)
		if req.ReceivedLen < len(aiRunes) {
			toSend := aiRunes[req.ReceivedLen:]
			conn.WriteMessage(websocket.TextMessage, []byte(string(toSend)))
		}
		conn.WriteMessage(websocket.TextMessage, []byte("[[END]]"))
		return
	}

	// History+轮询流式会话
	aiStreamSessionsLock.Lock()
	session, exists := aiStreamSessions[cacheKey]
	if !exists {
		session = &StreamSession{
			History: []rune{},
			Done:    make(chan struct{}),
		}
		aiStreamSessions[cacheKey] = session
		aiStreamSessionsLock.Unlock()

		// 查找最近N条历史消息
		var records []db.ChatRecord
		db.GetDB().Where("user_id = ?", user.ID).Order("created_at desc").Limit(10).Find(&records)
		var history []openai.ChatCompletionMessage
		for i := len(records) - 1; i >= 0; i-- {
			r := records[i]
			role := openai.ChatMessageRoleAssistant
			if r.IsUser {
				role = openai.ChatMessageRoleUser
			}
			history = append(history, openai.ChatCompletionMessage{
				Role:    role,
				Content: r.Content,
			})
		}
		history = append(history, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleSystem,
			Content: common.RolePrompt,
		})
		history = append(history, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: req.Content,
		})
		db.GetDB().Create(&db.ChatRecord{
			UserID:    user.ID,
			Content:   req.Content,
			IsUser:    true,
			CreatedAt: time.Now(),
			MsgID:     req.MsgID,
		})
		client := openai.NewClient(os.Getenv("OPENAI_API_KEY"))
		reqOpenAI := openai.ChatCompletionRequest{
			Model:    openai.GPT4,
			Messages: history,
			Stream:   true,
		}
		go func(sess *StreamSession) {
			var aiMsg string
			stream, err := client.CreateChatCompletionStream(c, reqOpenAI)
			if err != nil {
				close(sess.Done)
				return
			}
			defer stream.Close()
			for {
				response, err := stream.Recv()
				if err != nil {
					break
				}
				if len(response.Choices) > 0 {
					delta := response.Choices[0].Delta.Content
					if delta != "" {
						sess.History = append(sess.History, []rune(delta)...)
						aiMsg += delta
					}
				}
			}
			if aiMsg != "" {
				db.GetDB().Create(&db.ChatRecord{
					UserID:    user.ID,
					Content:   aiMsg,
					IsUser:    false,
					CreatedAt: time.Now(),
					MsgID:     req.MsgID,
				})
			}
			close(sess.Done)
			aiStreamSessionsLock.Lock()
			delete(aiStreamSessions, cacheKey)
			aiStreamSessionsLock.Unlock()
		}(session)
	} else {
		aiStreamSessionsLock.Unlock()
	}

	// 轮询补发新内容
	sentLen := req.ReceivedLen
	for {
		aiStreamSessionsLock.Lock()
		curLen := len(session.History)
		aiStreamSessionsLock.Unlock()
		if sentLen < curLen {
			toSend := session.History[sentLen:]
			err := conn.WriteMessage(websocket.TextMessage, []byte(string(toSend)))
			if err != nil {
				log.Printf("[AIWS] conn %s: WriteMessage error: %v", cacheKey, err)
				return
			}
			sentLen = curLen
		}
		select {
		case <-session.Done:
			conn.WriteMessage(websocket.TextMessage, []byte("[[END]]"))
			return
		case <-time.After(200 * time.Millisecond):
			// 继续轮询
		}
	}
}
