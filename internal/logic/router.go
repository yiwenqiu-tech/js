package logic

import (
	"context"
	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/memory"
	"jieyou-backend/internal/common"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	langopenai "github.com/tmc/langchaingo/llms/openai"
	"gorm.io/gorm"
	"jieyou-backend/internal/db"
)

const MaxChatPerDay = 10
const MaxTokenPerMsg = 500

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
	record := db.SignRecord{UserID: user.ID, Date: today, Type: "sign"}
	if err := db.GetDB().Create(&record).Error; err != nil {
		c.JSON(500, gin.H{"error": "db error"})
		return
	}
	c.JSON(200, gin.H{"message": "sign in success"})
}

// 破戒
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
	record := db.SignRecord{UserID: user.ID, Date: today, Type: "break"}
	if err := db.GetDB().Create(&record).Error; err != nil {
		c.JSON(500, gin.H{"error": "db error"})
		return
	}
	// 清空所有 type=sign 的签到记录
	db.GetDB().Where("user_id = ? AND type = ?", user.ID, "sign").Delete(&db.SignRecord{})
	c.JSON(200, gin.H{"message": "break success"})
}

// 日历
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

// 月排行榜
func MonthRankHandler(c *gin.Context) {
	month := time.Now().Format("2006-01")
	openid := c.Query("open_id")

	type Result struct {
		Nickname string
		Count    int64
		UserID   uint
		IsSelf   bool
		Rank     int
	}
	var results []Result
	db.GetDB().Table("sign_records").
		Select("users.nickname, sign_records.user_id, COUNT(*) as count").
		Joins("JOIN users ON users.id = sign_records.user_id").
		Where("sign_records.type = ? AND sign_records.date LIKE ?", "sign", month+"%").
		Group("user_id").
		Order("count DESC").
		Scan(&results)

	// 排名处理
	for i := range results {
		results[i].Rank = i + 1
	}

	var top10 []Result
	if len(results) > 10 {
		top10 = results[:10]
	} else {
		top10 = results
	}

	selfIdx := -1
	var self Result
	if openid != "" {
		var user db.User
		err := db.GetDB().Where("open_id = ?", openid).First(&user).Error
		if err == nil {
			for i, r := range results {
				if r.UserID == user.ID {
					selfIdx = i
					break
				}
			}
			if selfIdx >= 0 {
				if selfIdx < 10 {
					top10[selfIdx].IsSelf = true
				} else {
					self = results[selfIdx]
					self.IsSelf = true
					top10 = append(top10, self)
				}
			}
		}
	}

	c.JSON(200, gin.H{"rank": top10})
}

// 总排行榜
func TotalRankHandler(c *gin.Context) {
	openid := c.Query("open_id")
	type Result struct {
		Nickname string
		Count    int64
		UserID   uint
		IsSelf   bool
		Rank     int
	}
	var results []Result
	db.GetDB().Table("sign_records").
		Select("users.nickname, sign_records.user_id, COUNT(*) as count").
		Joins("JOIN users ON users.id = sign_records.user_id").
		Where("sign_records.type = ?", "sign").
		Group("user_id").
		Order("count DESC").
		Scan(&results)

	for i := range results {
		results[i].Rank = i + 1
	}

	var top10 []Result
	if len(results) > 10 {
		top10 = results[:10]
	} else {
		top10 = results
	}

	selfIdx := -1
	var self Result
	if openid != "" {
		var user db.User
		err := db.GetDB().Where("open_id = ?", openid).First(&user).Error
		if err == nil {
			for i, r := range results {
				if r.UserID == user.ID {
					selfIdx = i
					break
				}
			}
			if selfIdx >= 0 {
				if selfIdx < 10 {
					top10[selfIdx].IsSelf = true
				} else {
					self = results[selfIdx]
					self.IsSelf = true
					top10 = append(top10, self)
				}
			}
		}
	}

	c.JSON(200, gin.H{"rank": top10})
}

// AI 聊天接口
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
		langopenai.WithToken(common.HunyuanToekn),
		langopenai.WithModel(common.HunyuanModel),
		langopenai.WithBaseURL(common.HunyuanBaseUrl))
	chain := chains.NewConversation(llm, chatMemory)
	resp, err := chains.Run(ctx, chain, req.Content, chains.WithMaxTokens(600))
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
	c.JSON(200, gin.H{"records": records})
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
