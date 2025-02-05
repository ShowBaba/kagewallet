package jobs

import (
	"encoding/json"
	"fmt"

	"go.uber.org/zap"
	"kagewallet/bot"
	"kagewallet/common"
	"kagewallet/database"
	log "kagewallet/logging"
)

func ListenForNotifications() {
	sub := database.RedisSubscribe(common.RedisNotificationChannelKey)
	defer sub.Close()

	ch := sub.Channel()
	for msg := range ch {
		go func(notificationKey string) {
			fmt.Println("Notification received:", notificationKey)
			notificationJSON, err := database.HGet(common.RedisNotificationKey, notificationKey)
			if err != nil {
				log.Error("failed to fetch notification", zap.String("key", notificationKey), zap.Error(err))
				return
			}

			var notification common.Notification
			err = json.Unmarshal([]byte(notificationJSON), &notification)
			if err != nil {
				log.Error("failed to unmarshal notification", zap.String("key", notificationKey), zap.Error(err))
				return
			}
			if notification.Status == "delivered" {
				return
			}

			chatRedisData, err := database.HGet(common.RedisActiveChatsKey, notification.To)
			if err != nil {
				log.Error("failed to fetch chat data", zap.Error(err))
				return
			}

			fmt.Println("chat data: ", chatRedisData)

			var chatData common.TelegramChatMetadata
			if err := json.Unmarshal([]byte(chatRedisData), &chatData); err != nil {
				log.Error("failed to unmarshal chat data", zap.Error(err))
				return
			}

			err = bot.SendTelegramUserMessage(chatData.ChatID, notification.Payload.(string))
			if err != nil {
				log.Error("failed to send notification", zap.String("key", notificationKey), zap.Error(err))
				notification.Status = "failed"
			} else {
				notification.Status = "delivered"
			}

			updatedNotification, _ := json.Marshal(notification)
			err = database.HSet(common.RedisNotificationKey, notificationKey, updatedNotification)
			if err != nil {
				log.Error("failed to update notification status", zap.String("key", notificationKey), zap.Error(err))
			}
		}(msg.Payload)
	}
}
