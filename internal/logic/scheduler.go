package logic

import (
	"log"
	"time"

	"jieyou-backend/internal/db"
)

// CheckAndSendReminders 检查并发送打卡提醒
func CheckAndSendReminders() {
	log.Println("开始检查用户打卡状态...")

	// 检查数据库是否已初始化
	if db.GetDB() == nil {
		log.Println("数据库未初始化，跳过打卡提醒检查")
		return
	}

	today := time.Now().Format("2006-01-02")

	// 获取所有已授权订阅的用户
	var subscriptions []db.Subscription
	if err := db.GetDB().Preload("User").Where("is_auth = ?", true).
		Find(&subscriptions).Error; err != nil {
		log.Printf("获取订阅用户列表失败: %v", err)
		return
	}

	reminderCount := 0
	successCount := 0

	for _, subscription := range subscriptions {
		user := subscription.User

		// 检查用户今天是否已经打卡
		var signCount int64
		if err := db.GetDB().Model(&db.SignRecord{}).
			Where("user_id = ? AND date = ? AND type = ?", user.ID, today, "sign").
			Count(&signCount).Error; err != nil {
			log.Printf("检查用户 %s 打卡状态失败: %v", user.Nickname, err)
			continue
		}

		// 检查用户今天是否已经破戒
		var breakCount int64
		if err := db.GetDB().Model(&db.SignRecord{}).
			Where("user_id = ? AND date = ? AND type = ?", user.ID, today, "break").
			Count(&breakCount).Error; err != nil {
			log.Printf("检查用户 %s 破戒状态失败: %v", user.Nickname, err)
			continue
		}

		// 如果既没有打卡也没有破戒，发送提醒
		if signCount == 0 && breakCount == 0 {
			reminderCount++
			if err := SendSignInReminder(user.OpenID, user.Nickname); err != nil {
				log.Printf("发送提醒给用户 %s 失败: %v", user.Nickname, err)
				// 如果发送失败，可能是用户取消了订阅，更新数据库状态
				subscription.IsAuth = false
				db.GetDB().Save(&subscription)
				log.Printf("用户 %s 可能已取消订阅，更新状态", user.Nickname)
			} else {
				successCount++
				log.Printf("成功发送提醒给用户: %s", user.Nickname)

				// 发送成功后，将授权状态设为false，因为微信订阅消息是一次性的
				subscription.IsAuth = false
				db.GetDB().Save(&subscription)
				log.Printf("用户 %s 的订阅已使用，需要重新授权", user.Nickname)
			}
		}
	}

	log.Printf("打卡提醒检查完成: 需要提醒 %d 人，成功发送 %d 人", reminderCount, successCount)
}

// StartScheduler 启动定时任务
func StartScheduler() {
	log.Println("启动定时任务调度器...")

	// 设置定时任务，每天晚上8:30检查
	go func() {
		for {
			now := time.Now()
			next := time.Date(now.Year(), now.Month(), now.Day(), 20, 30, 0, 0, now.Location())

			// 如果今天已经过了8:30，设置为明天8:30
			if now.After(next) {
				next = next.Add(24 * time.Hour)
			}

			// 等待到下次执行时间
			sleepDuration := next.Sub(now)
			log.Printf("下次打卡提醒检查时间: %s (等待 %v)", next.Format("2006-01-02 15:04:05"), sleepDuration)
			time.Sleep(sleepDuration)

			// 执行检查
			CheckAndSendReminders()
		}
	}()
}
