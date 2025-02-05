package common

import "fmt"

const (
	RedisPasswordSetupKey             = "passwordSetup:%d"
	RedisWithdrawSetupKey             = "withdrawSetup:%d"
	RedisWithdrawalAmountSetupKey     = "withdrawalAmountSetup:%d"
	RedisSelectedBankKey              = "selectedBank:%d"
	RedisBankAccountNumberKey         = "bankAccountNumber:%d"
	RedisSetBankAccountNumberKey      = "setBankAccountNumber:%d"
	RedisConfirmWithdrawalPasswordKey = "confirmWithdrawalPassword:%d"
	RedisEmailSetupKey                = "emailSetup:%d"
	RedisSearchBankKey                = "searchBank:%d"
	RedisAssetSelectionKey            = "assetSelection:%d"
	RedisActiveChatsKey               = "activeChats"
	RedisNotificationKey              = "notifications"
	RedisNotificationChannelKey       = "notificationChannel"
	RedisMonnifyToken                 = "monnifyToken"
	NairaAssetID                      = "0f0a0c3c-9a0a-4ec4-9be0-3ddea69327b3"
	WithdrawalFee                     = 100
)

var GenerateRedisDeleteKeyPattern = func(chatId int64) string {
	return fmt.Sprintf("passwordSetup:%d|emailSetup:%d", chatId, chatId)
}
