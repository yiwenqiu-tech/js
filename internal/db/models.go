package db

import (
	"time"
)

type User struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	OpenID    string    `gorm:"size:64;uniqueIndex" json:"open_id"` // 微信openid
	Nickname  string    `gorm:"size:32" json:"nickname"`
	CreatedAt time.Time `json:"created_at"`
}

type SignRecord struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index" json:"user_id"`
	Date      string    `gorm:"size:10;index" json:"date"` // yyyy-mm-dd
	Type      string    `gorm:"size:8" json:"type"`        // sign/break
	CreatedAt time.Time `json:"created_at"`
}

// ChatRecord 聊天记录表
// is_user: true 表示用户发言，false 表示AI回复
// content: 聊天内容
// created_at: 创建时间
type ChatRecord struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index" json:"user_id"`
	Content   string    `gorm:"type:text" json:"content"`
	IsUser    bool      `json:"is_user"`
	CreatedAt time.Time `json:"created_at"`
}
