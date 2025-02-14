package bot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ShowBaba/kagewallet/common"
	"github.com/ShowBaba/kagewallet/database"
	"github.com/ShowBaba/kagewallet/helpers"
	log "github.com/ShowBaba/kagewallet/logging"
	"github.com/ShowBaba/kagewallet/repositories"
	"github.com/ShowBaba/kagewallet/services"
	"github.com/ShowBaba/kagewallet/tmpl"
	"github.com/dustin/go-humanize"
	tgApi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	CommandStart              = "/start"
	CommandHelp               = "/help"
	CommandCommand            = "/command"
	CommandCommands           = "/commands"
	CommandSetPassword        = "/set_password"
	CommandResetPassword      = "/reset_password"
	CommandRefresh            = "/refresh"
	CommandRate               = "/rate"
	CommandRates              = "/rates"
	CommandGenerate           = "/generate"
	CommandGenerateAddress    = "/generate_address"
	CommandSell               = "/sell"
	CommandBalances           = "/balances"
	CommandBalance            = "/balance"
	CommandTransactions       = "/transactions"
	CommandTransaction        = "/transaction"
	CommandTransactionHistory = "/transaction_history"
	CommandWithdraw           = "/withdraw"
)

const (
	fetchTransactionLimit = 2
	fetchBankLimit        = 10
)

var (
	mu                 = &sync.RWMutex{}
	userRepo           *repositories.UserRepository
	telegramRepo       *repositories.TelegramRepository
	telegramCmdLogRepo *repositories.TelegramCommandLogRepository
	rateRepo           *repositories.RateRepository
	assetRepo          *repositories.AssetRepository
	authService        *services.AuthService
	addressRepo        *repositories.AddressRepository
	addressService     *services.AddressService
	rateService        *services.RateService
	walletRepo         *repositories.WalletRepository
	transactionRepo    *repositories.TransactionRepository
	withdrawalRepo     *repositories.WithdrawalRepository
	walletService      *services.WalletService
	transactionService *services.TransactionService
	monnifyService     *services.MonnifyService
	withdrawalService  *services.WithdrawalService
	ctx, _             = context.WithCancel(context.Background())
)

type TelegramBot struct {
	Api *tgApi.BotAPI
}

type TelegramMessage struct {
	Text        string
	User        int64
	ParseMode   string
	ReplyMarkup interface{}
	File        tgApi.FileBytes
}

var Telegram *TelegramBot

func NewTelegramBot(TOKEN string, db *gorm.DB) (*TelegramBot, error) {
	bot, err := tgApi.NewBotAPI(TOKEN)
	if err != nil {
		return nil, err
	}

	bot.Debug = false

	tBot := TelegramBot{Api: bot}

	commands := []tgApi.BotCommand{
		{Command: CommandStart, Description: "Start the bot and see the list of commands"},
		{Command: CommandSell, Description: "Sell cryptocurrency and receive payment in fiat"},
		{Command: CommandHelp, Description: "See a list of available commands and their descriptions"},
		{Command: CommandSetPassword, Description: "Set a new password for your account"},
		{Command: CommandResetPassword, Description: "Reset your password if forgotten"},
		{Command: CommandRefresh, Description: "Refresh your session and update data"},
		{Command: CommandRate, Description: "Get the current exchange rate"},
		{Command: CommandBalance, Description: "Check your balance for a specific asset"},
		{Command: CommandTransactions, Description: "View your complete transaction history"},
		{Command: CommandWithdraw, Description: "Withdraw funds to your bank account"},
	}

	setCommandsConfig := tgApi.NewSetMyCommands(commands...)
	_, err = bot.Request(setCommandsConfig)
	if err != nil {
		panic(err)
	}

	Telegram = &tBot

	userRepo = repositories.NewUserRepository(db)
	telegramRepo = repositories.NewTelegramRepository(db)
	telegramCmdLogRepo = repositories.NewTelegramCommandLogRepository(db)
	rateRepo = repositories.NewRateRepository(db)
	rateService = services.NewRateService(rateRepo)
	assetRepo = repositories.NewAssetRepository(db)
	addressRepo = repositories.NewAddressRepository(db)
	addressService = services.NewAddressService(userRepo, addressRepo, assetRepo)
	authService = services.NewAuthService(userRepo)
	walletRepo = repositories.NewWalletRepository(db)
	walletService = services.NewWalletService(walletRepo, assetRepo)
	transactionRepo = repositories.NewTransactionRepository(db)
	withdrawalRepo = repositories.NewWithdrawalRepository(db)
	transactionService = services.NewTransactionService(userRepo, transactionRepo)
	monnifyService = services.NewMonnifyService()
	withdrawalService = services.NewWithdrawalService(monnifyService, withdrawalRepo, walletRepo)
	return &tBot, err
}

func (tb *TelegramBot) ListenForUpdates() {
	updateConfig := tgApi.NewUpdate(0)
	updateConfig.Timeout = 60

	updates := tb.Api.GetUpdatesChan(updateConfig)

	for {
		select {
		case <-ctx.Done():
			return
		case update := <-updates:
			handleUpdate(update)
		}
	}
}

func (tb *TelegramBot) SendUserMessage(message TelegramMessage) error {
	mode := "HTML"
	if message.ParseMode != "" {
		mode = message.ParseMode
	}

	msg := tgApi.NewMessage(message.User, message.Text)
	msg.ParseMode = mode
	msg.ReplyMarkup = message.ReplyMarkup

	if _, err := tb.Api.Send(msg); err != nil {
		log.Error("error sending message", zap.Error(err))
		return err
	}

	fmt.Println("Message sent successfully")
	return nil
}

func (tb *TelegramBot) SendLoader(chatId int64) error {
	chatAction := tgApi.NewChatAction(chatId, tgApi.ChatTyping)
	if _, err := tb.Api.Request(chatAction); err != nil {
		log.Error("error sending chat action", zap.Error(err))
		return err
	}
	return nil
}

type TelegramMessageEdit struct {
	ChatID      int64
	MessageID   int
	NewText     string
	ReplyMarkup *tgApi.InlineKeyboardMarkup
	ParseMode   string
}

func (tb *TelegramBot) EditMessage(params TelegramMessageEdit) error {
	editConfig := tgApi.NewEditMessageText(params.ChatID, params.MessageID, params.NewText)

	if params.ParseMode != "" {
		editConfig.ParseMode = params.ParseMode
	} else {
		editConfig.ParseMode = "HTML"
	}
	editConfig.ReplyMarkup = params.ReplyMarkup

	if _, err := tb.Api.Send(editConfig); err != nil {
		return err
	}

	fmt.Println("Message edited successfully")
	return nil
}

func (tb *TelegramBot) TelegramMessageEdit(chatID int64, messageID int, newText string, replyMarkup *tgApi.InlineKeyboardMarkup) error {
	editConfig := tgApi.NewEditMessageText(chatID, messageID, newText)
	editConfig.ParseMode = "HTML"
	editConfig.ReplyMarkup = replyMarkup

	if _, err := tb.Api.Send(editConfig); err != nil {
		return err
	}

	fmt.Println("Message edited successfully")
	return nil
}

func (tb *TelegramBot) SendCallbackResponse(response common.TelegramCallbackResponse) error {
	callbackConfig := tgApi.CallbackConfig{
		CallbackQueryID: response.CallbackQueryID,
		Text:            response.Text,
		ShowAlert:       response.ShowAlert,
		URL:             response.URL,
		CacheTime:       response.CacheTime,
	}

	if _, err := tb.Api.Request(callbackConfig); err != nil {
		return err
	}

	fmt.Println("Callback response sent successfully")
	return nil
}

func (tb *TelegramBot) Webhook(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Error("failed to read body: %v", zap.Error(err))
	}

	var update tgApi.Update
	if err := json.Unmarshal(body, &update); err != nil {
		log.Error("failed to unmarshal body: %v", zap.Error(err))
		return
	}

	handleUpdate(update)
}

func handleUpdate(update tgApi.Update) {
	// fmt.Printf("[Received Message] From: %s, Text: %s", update.Message.From.UserName, update.Message.Text)

	updateRecord(update)

	switch {
	case update.Message != nil:
		err := handleMessage(update.Message)
		if err != nil {
			err = sendErrorMessage(update.Message.Chat.ID)
			if err != nil {
				log.Error("error sending error message", zap.Error(err))

			}
		}
	case update.CallbackQuery != nil:
		err := handleCallback(update.CallbackQuery)
		if err != nil {
			err = sendErrorMessage(update.Message.Chat.ID)
			if err != nil {
				log.Error("error sending error message", zap.Error(err))
			}
		}
	}

}

func sendErrorMessage(chatId int64) error {
	text, _ := helpers.FormatHTML(nil, tmpl.ErrorMessage)
	return Telegram.SendUserMessage(TelegramMessage{Text: text, User: int64(chatId)})
}

func handleMessage(message *tgApi.Message) error {
	var (
		text       = message.Text
		chat       = message.Chat
		telegramId = message.From.ID
	)

	if strings.HasPrefix(text, "/") {
		switch text {
		case CommandStart:
			text, _ := helpers.FormatHTML(nil, tmpl.WelcomeMessage)

			return Telegram.SendUserMessage(TelegramMessage{Text: text, User: chat.ID})
		case CommandHelp, CommandCommand, CommandCommands:
			text, _ := helpers.FormatHTML(nil, tmpl.Commands)

			return Telegram.SendUserMessage(TelegramMessage{Text: text, User: chat.ID})
		case CommandSetPassword:
			user, err := telegramRepo.FindUserByTelegramID(int(telegramId)) // user will always exist from the updateRecord middleware
			if err != nil {
				log.Error("error fetching user by telegram id", zap.Error(err))
				return sendErrorMessage(message.Chat.ID)
			}
			hasSetPassword, err := userRepo.HasSetPassword(user.ID)
			if err != nil {
				log.Error("error checking if user has set password", zap.Error(err))
				return sendErrorMessage(message.Chat.ID)
			}
			if hasSetPassword {
				text, _ := helpers.FormatHTML(nil, tmpl.PasswordAlreadySet)
				return Telegram.SendUserMessage(TelegramMessage{Text: text, User: chat.ID})
			}

			err = database.SetRedisKey(fmt.Sprintf(common.RedisPasswordSetupKey, chat.ID), "true", 0)
			if err != nil {
				log.Error("error setting redis key", zap.Error(err))

			}

			text, _ := helpers.FormatHTML(nil, tmpl.PasswordPrompt)
			return Telegram.SendUserMessage(TelegramMessage{Text: text, User: chat.ID})
		case CommandRefresh:
			err := database.DeleteRedisKeysByPattern(common.GenerateRedisDeleteKeyPattern(chat.ID))
			if err != nil {
				log.Error("error refreshing chat state", zap.Error(err))
				text, _ := helpers.FormatHTML(nil, tmpl.RefreshChatFailed)
				return Telegram.SendUserMessage(TelegramMessage{Text: text, User: chat.ID})
			}

			text, _ := helpers.FormatHTML(nil, tmpl.RefreshChatSuccess)
			return Telegram.SendUserMessage(TelegramMessage{Text: text, User: chat.ID})
		case CommandRate, CommandRates:
			rate, err := rateService.GetCurrentRate()
			if err != nil {
				log.Error("error fetching rates", zap.Error(err))
				return sendErrorMessage(message.Chat.ID)
			}

			text := fmt.Sprintf(
				"📈 *Current Exchange Rate* 📉\n\n"+
					"💵 *1 USD = %v* Naira\n\n"+
					"⏳ *Last Updated:* %s\n\n"+
					"🔄 Rates may fluctuate. Stay updated!",
				rate.Rate,
				time.Now().Format("02 Jan 2006, 03:04 PM"),
			)

			return Telegram.SendUserMessage(TelegramMessage{Text: text, User: chat.ID, ParseMode: "markdown"})
		case CommandGenerate, CommandGenerateAddress, CommandSell:
			if err := Telegram.SendLoader(chat.ID); err != nil {
				log.Error("error sending loader", zap.Error(err))
				return sendErrorMessage(chat.ID)
			}
			assets, err := assetRepo.GetActiveAssets()
			if err != nil {
				log.Error("error fetching active assets", zap.Error(err))
				text := "Failed to fetch assets. Please try again later."
				return Telegram.SendUserMessage(TelegramMessage{Text: text, User: chat.ID})
			}

			if len(assets) == 0 {
				text := "No active assets are available at the moment."
				return Telegram.SendUserMessage(TelegramMessage{Text: text, User: chat.ID})
			}

			buttons := make([][]tgApi.InlineKeyboardButton, len(assets))
			for i, asset := range assets {
				buttons[i] = []tgApi.InlineKeyboardButton{
					{
						Text: func() string {
							s := fmt.Sprintf("%s", strings.ToUpper(asset.Symbol))
							if asset.Standard != "" {
								s = fmt.Sprintf("%s (%s)", strings.ToUpper(asset.Symbol), strings.ToUpper(asset.Standard))
							}
							return s
						}(),
						CallbackData: helpers.StrPtr(fmt.Sprintf("generate_address:%s", asset.ID.String())),
					},
				}
			}

			replyMarkup := tgApi.InlineKeyboardMarkup{InlineKeyboard: buttons}
			text := "Please select a crypto to sell:"
			return Telegram.SendUserMessage(TelegramMessage{
				Text:        text,
				User:        chat.ID,
				ReplyMarkup: replyMarkup,
				ParseMode:   "markdown",
			})
		case CommandBalance, CommandBalances:
			// TODO: ask for password
			user, err := telegramRepo.FindUserByTelegramID(int(telegramId))
			if err != nil {
				log.Error("error fetching user by telegram id", zap.Error(err))
				return sendErrorMessage(message.Chat.ID)
			}
			wallet, err := walletService.GetUserWalletsData(user.ID.String())
			if err != nil {
				log.Error("failed to fetch user wallets", zap.Error(err))
				text := "Sorry, we couldn't retrieve your wallet balances at this time. Please try again later."
				return Telegram.SendUserMessage(TelegramMessage{Text: text, User: chat.ID})
			}
			var m strings.Builder
			m.WriteString("💰 *Your Wallet Balance:* 💼\n\n")

			if wallet == nil {
				m.WriteString("🔴 *₦0.00* – No funds available.")
			} else {
				m.WriteString(fmt.Sprintf(
					"🟢 *₦%s* – Available Balance",
					humanize.Commaf(wallet.Balance),
				))
			}

			m.WriteString("\n\n📌 Use /sell to sell crypto or /withdraw to cash out.")

			footer, err := getFooter()
			if err != nil {
				log.Error("error getting footer", zap.Error(err))
				return sendErrorMessage(message.Chat.ID)
			}
			m.WriteString(footer)

			return Telegram.SendUserMessage(TelegramMessage{Text: m.String(), User: chat.ID, ParseMode: "Markdown"})
		case CommandTransactions, CommandTransactionHistory, CommandTransaction:
			// TODO: ask for password
			if err := Telegram.SendLoader(chat.ID); err != nil {
				log.Error("error sending loader", zap.Error(err))
				return sendErrorMessage(chat.ID)
			}
			user, err := telegramRepo.FindUserByTelegramID(int(telegramId))
			if err != nil {
				log.Error("error fetching user by telegram id", zap.Error(err))
				return sendErrorMessage(message.Chat.ID)
			}

			var (
				page   = 1
				offset = (page - 1) * fetchTransactionLimit
			)

			transactions, err := transactionService.FetchUserTransactions(user.ID.String(), fetchTransactionLimit, offset)
			if err != nil {
				log.Error("failed to fetch user transactions", zap.Error(err))
				return sendErrorMessage(message.Chat.ID)
			}

			totalTransactions, err := transactionService.FetchUserTransactionCount(user.ID.String())
			if err != nil {
				log.Error("failed to count user transactions", zap.Error(err))
				return sendErrorMessage(message.Chat.ID)
			}

			var m strings.Builder
			totalPages := (totalTransactions + fetchTransactionLimit - 1) / fetchTransactionLimit
			m.WriteString(fmt.Sprintf("*Your Transaction History (Page %d of %d):*\n\n", page, totalPages))

			for _, tx := range transactions {
				nairaBalance := humanize.Commaf(tx.Amount * tx.Rate)
				usdBalance := humanize.Commaf(tx.AmountUSD)
				m.WriteString(fmt.Sprintf(
					"📜 *Transaction Details*\n"+
						"  ━━━━━━━━━━━━━━  \n"+
						"🔖 *Type:* %s\n"+
						"🆔 *Transaction Reference:* `%s`\n"+
						"💰 *Currency:* %s\n"+
						"📊 *Amount:* `%.2f %s`\n"+
						"🔵 *Status:* %s\n"+
						"🔂 *Confirmations:* %d\n"+
						"📅 *Date:* %s\n"+
						"💹 *Rate:* `₦%.2f`\n"+
						"💵 *Amount in Naira:* `₦%s`\n"+
						"💲 *Amount in USD:* `$%s`\n"+
						"  ━━━━━━━━━━━━━━  \n",
					tx.Type,
					tx.Reference,
					func() string {
						s := fmt.Sprintf("%s", strings.ToUpper(tx.AssetSymbol))
						if tx.AssetStandard != "" {
							s = fmt.Sprintf("%s (%s)", strings.ToUpper(tx.AssetSymbol), strings.ToUpper(tx.AssetStandard))
						}
						return s
					}(),
					tx.Amount,
					tx.AssetSymbol,
					tx.Status,
					tx.Confirmations,
					tx.CreatedAt.Format("02 Jan 2006, 03:04 PM"),
					tx.Rate,
					nairaBalance,
					usdBalance,
				))
				m.WriteString("\n")
			}

			var buttons [][]tgApi.InlineKeyboardButton

			if totalTransactions > page*fetchTransactionLimit {
				nextPageButton := tgApi.NewInlineKeyboardButtonData(
					fmt.Sprintf("Next Page (%d)", page+1),
					fmt.Sprintf("transactions_page:%d", page+1),
				)
				buttons = append(buttons, []tgApi.InlineKeyboardButton{nextPageButton})
				replyMarkup := tgApi.InlineKeyboardMarkup{InlineKeyboard: buttons}
				return Telegram.SendUserMessage(TelegramMessage{
					Text:        m.String(),
					User:        chat.ID,
					ReplyMarkup: replyMarkup,
					ParseMode:   "markdown",
				})
			} else {
				return Telegram.SendUserMessage(TelegramMessage{
					Text:      m.String(),
					User:      chat.ID,
					ParseMode: "markdown",
				})
			}
		case CommandWithdraw:
			if err := Telegram.SendLoader(chat.ID); err != nil {
				log.Error("error sending loader", zap.Error(err))
				return sendErrorMessage(chat.ID)
			}
			user, err := telegramRepo.FindUserByTelegramID(int(telegramId))
			if err != nil {
				log.Error("error fetching user by telegram id", zap.Error(err))
				return sendErrorMessage(message.Chat.ID)
			}
			wallet, err := walletService.GetUserWalletsData(user.ID.String())
			if err != nil {
				log.Error("failed to fetch user wallets", zap.Error(err))
				text := "Sorry, we couldn't retrieve your wallet balances at this time. Please try again later."
				return Telegram.SendUserMessage(TelegramMessage{Text: text, User: chat.ID})
			}
			var m strings.Builder
			m.WriteString("💰 *Your Wallet Balance:* \n\n")

			if wallet == nil || wallet.Balance == 0 {
				m.WriteString("🚨 *₦0.00*\n\n")
				m.WriteString("😕 Oops! You don’t have enough balance to withdraw.\n\n")
				return Telegram.SendUserMessage(TelegramMessage{Text: m.String(), User: chat.ID, ParseMode: "Markdown"})
			}

			m.WriteString(fmt.Sprintf("💵 *₦%s*\n\n", humanize.Commaf(wallet.Balance)))

			err = database.SetRedisKey(fmt.Sprintf(common.RedisWithdrawSetupKey, chat.ID), "true", 0)
			if err != nil {
				log.Error("error setting redis key", zap.Error(err))
			}

			buttons := [][]tgApi.InlineKeyboardButton{
				{
					{Text: "💸 Withdraw All", CallbackData: helpers.StrPtr("withdraw_all")},
				},
			}

			replyMarkup := tgApi.InlineKeyboardMarkup{InlineKeyboard: buttons}

			m.WriteString("🔹 Click the *Withdraw All* button or enter an amount to withdraw.\n\n")
			m.WriteString(fmt.Sprintf("⚠️ *A withdrawal fee of ₦%v applies.*", common.WithdrawalFee))

			return Telegram.SendUserMessage(TelegramMessage{Text: m.String(), User: chat.ID,
				ReplyMarkup: replyMarkup, ParseMode: "Markdown"})
		default:
			text, _ := helpers.FormatHTML(nil, tmpl.Commands)
			return Telegram.SendUserMessage(TelegramMessage{Text: text, User: chat.ID})
		}
	}

	if state, _ := database.GetRedisKey(fmt.Sprintf(common.RedisPasswordSetupKey, chat.ID)); state == "true" {
		if len(text) < 8 {
			text, _ := helpers.FormatHTML(nil, tmpl.PasswordTooShort)
			return Telegram.SendUserMessage(TelegramMessage{Text: text, User: chat.ID})
		}

		user, err := telegramRepo.FindUserByTelegramID(int(telegramId))
		if err != nil {
			log.Error("error fetching user by telegram id", zap.Error(err))
			return sendErrorMessage(message.Chat.ID)
		}

		err = authService.SetPassword(common.SetPasswordInput{
			UserID:   user.ID.String(),
			Password: text,
		})
		if err != nil {
			log.Error("error setting user password", zap.Error(err))
			text, _ := helpers.FormatHTML(nil, tmpl.PasswordSetFailed)
			return Telegram.SendUserMessage(TelegramMessage{Text: text, User: chat.ID})
		}

		err = database.DeleteRedisKey(fmt.Sprintf(common.RedisPasswordSetupKey, chat.ID))
		if err != nil {
			log.Error("error deleting redis key", zap.Error(err))
		}

		text, _ := helpers.FormatHTML(nil, tmpl.PasswordSetSuccess)
		err = Telegram.SendUserMessage(TelegramMessage{Text: text, User: chat.ID})
		if err != nil {
			log.Error("error sending success message", zap.Error(err))
			return sendErrorMessage(message.Chat.ID)
		}

		err = database.SetRedisKey(fmt.Sprintf(common.RedisEmailSetupKey, chat.ID), "true", 0)
		if err != nil {
			log.Error("error setting redis key", zap.Error(err))
		}

		text, _ = helpers.FormatHTML(nil, tmpl.EmailPrompt)
		return Telegram.SendUserMessage(TelegramMessage{Text: text, User: chat.ID})
	}

	if state, _ := database.GetRedisKey(fmt.Sprintf(common.RedisEmailSetupKey, chat.ID)); state == "true" {
		if !helpers.IsValidEmail(text) {
			text, _ := helpers.FormatHTML(nil, tmpl.InvalidEmail)
			return Telegram.SendUserMessage(TelegramMessage{Text: text, User: chat.ID})
		}

		user, err := telegramRepo.FindUserByTelegramID(int(telegramId))
		if err != nil {
			log.Error("error fetching user by telegram id", zap.Error(err))
			return sendErrorMessage(chat.ID)
		}

		err = userRepo.UpdateField(user.ID, "email", text)
		if err != nil {
			log.Error("error updating user email", zap.Error(err))
			text, _ := helpers.FormatHTML(nil, tmpl.EmailUpdateFailed)
			return Telegram.SendUserMessage(TelegramMessage{Text: text, User: chat.ID})
		}

		err = database.DeleteRedisKey(fmt.Sprintf(common.RedisEmailSetupKey, chat.ID))
		if err != nil {
			log.Error("error deleting redis key", zap.Error(err))
		}

		text, _ := helpers.FormatHTML(nil, tmpl.EmailUpdateSuccess)
		return Telegram.SendUserMessage(TelegramMessage{Text: text, User: chat.ID})
	}

	if state, _ := database.GetRedisKey(fmt.Sprintf(common.RedisWithdrawSetupKey, chat.ID)); state == "true" {
		if err := Telegram.SendLoader(chat.ID); err != nil {
			log.Error("error sending loader", zap.Error(err))
			return sendErrorMessage(chat.ID)
		}
		amount, err := validateAmount(text)
		if err != nil {
			log.Error("error getting withdrawal amount", zap.Error(err))
			return Telegram.SendUserMessage(TelegramMessage{Text: err.Error(), User: chat.ID})
		}
		user, err := telegramRepo.FindUserByTelegramID(int(telegramId))
		if err != nil {
			log.Error("error fetching user by telegram id", zap.Error(err))
			return sendErrorMessage(message.Chat.ID)
		}
		wallet, err := walletService.GetUserWalletsData(user.ID.String())
		if err != nil {
			log.Error("failed to fetch user wallets", zap.Error(err))
			text := "Sorry, we couldn't retrieve your wallet balances at this time. Please try again later."
			return Telegram.SendUserMessage(TelegramMessage{Text: text, User: chat.ID})
		}
		if amount > wallet.Balance {
			var m strings.Builder
			m.WriteString("🚨 *Insufficient Balance!* 🚨\n\n")
			m.WriteString("💰 *Your Wallet Balance:*\n")
			m.WriteString(fmt.Sprintf(
				"💵 *₦%s*\n\n",
				humanize.Commaf(wallet.Balance),
			))

			buttons := [][]tgApi.InlineKeyboardButton{
				{
					{Text: "💸 Withdraw All", CallbackData: helpers.StrPtr("withdraw_all")},
				},
			}

			replyMarkup := tgApi.InlineKeyboardMarkup{InlineKeyboard: buttons}

			m.WriteString("🔹 Click the *Withdraw All* button or enter an amount to withdraw.\n\n")
			m.WriteString(fmt.Sprintf("⚠️ *A withdrawal fee of ₦%v applies.*", common.WithdrawalFee))

			return Telegram.SendUserMessage(TelegramMessage{Text: m.String(), User: chat.ID,
				ReplyMarkup: replyMarkup, ParseMode: "Markdown"})
		} else {
			err := database.DeleteRedisKey(fmt.Sprintf(common.RedisWithdrawSetupKey, chat.ID))
			if err != nil {
				log.Error("error deleting redis key", zap.Error(err))
			}
			err = database.SetRedisKey(fmt.Sprintf(common.RedisWithdrawalAmountSetupKey, chat.ID), fmt.Sprintf(`%v`, amount), 0)
			if err != nil {
				log.Error("error setting redis key", zap.Error(err))
			}
			return getBanks(chat.ID)
		}
	}

	if state, _ := database.GetRedisKey(fmt.Sprintf(common.RedisSearchBankKey, chat.ID)); state == "true" {
		if err := Telegram.SendLoader(chat.ID); err != nil {
			log.Error("error sending loader", zap.Error(err))
			return sendErrorMessage(chat.ID)
		}
		page := 1

		paginatedBanks, totalPages, err := withdrawalService.SearchBank(text, page, fetchBankLimit)
		if err != nil {
			log.Error("error searching bank", zap.Error(err))
			return sendErrorMessage(chat.ID)
		}
		var buttons [][]tgApi.InlineKeyboardButton
		for _, bank := range paginatedBanks {
			buttons = append(buttons, []tgApi.InlineKeyboardButton{
				{
					Text:         bank.Name,
					CallbackData: helpers.StrPtr(fmt.Sprintf("select_bank:%s", bank.Code)),
				},
			})
		}
		if page < totalPages {
			buttons = append(buttons, []tgApi.InlineKeyboardButton{
				{
					Text:         "Next Page ➡️",
					CallbackData: helpers.StrPtr(fmt.Sprintf("search_banks_page:%d:%s", page+1, text)),
				},
			})
		}
		buttons = append(buttons, []tgApi.InlineKeyboardButton{
			{
				Text:         "Search Bank 🔍",
				CallbackData: helpers.StrPtr("search_bank"),
			},
		})
		replyMarkup := tgApi.InlineKeyboardMarkup{InlineKeyboard: buttons}
		err = database.DeleteRedisKey(fmt.Sprintf(common.RedisSearchBankKey, chat.ID))
		if err != nil {
			log.Error("error deleting redis key", zap.Error(err))
		}
		return Telegram.SendUserMessage(TelegramMessage{
			Text:        "Please select your bank:",
			User:        chat.ID,
			ReplyMarkup: replyMarkup,
		})
	}

	if state, _ := database.GetRedisKey(fmt.Sprintf(common.RedisSetBankAccountNumberKey, chat.ID)); state == "true" {
		if err := Telegram.SendLoader(chat.ID); err != nil {
			log.Error("error sending loader", zap.Error(err))
			return sendErrorMessage(chat.ID)
		}

		redisKey := fmt.Sprintf(common.RedisSelectedBankKey, chat.ID)
		bankCode, err := database.GetRedisKey(redisKey)
		if err != nil {
			log.Error("error validating account", zap.Error(err))
			return sendErrorMessage(chat.ID)
		}
		amountRedisKey := fmt.Sprintf(common.RedisWithdrawalAmountSetupKey, chat.ID)
		withdrawalAmtStr, err := database.GetRedisKey(amountRedisKey)
		if err != nil {
			log.Error("error validating account", zap.Error(err))
			return sendErrorMessage(chat.ID)
		}
		withdrawalAmt, err := strconv.ParseFloat(withdrawalAmtStr, 64)
		if err != nil {
			log.Error("error validating account", zap.Error(err))
			return sendErrorMessage(chat.ID)
		}
		accountNumber := text
		err = database.SetRedisKey(fmt.Sprintf(common.RedisBankAccountNumberKey, chat.ID), accountNumber, 0)
		if err != nil {
			log.Error("error setting redis key", zap.Error(err))

		}

		bankData, err := withdrawalService.GetBankByCode(bankCode)
		if err != nil {
			log.Error("error fetching banks", zap.Error(err))
			return Telegram.SendUserMessage(TelegramMessage{
				Text: "Failed to load banks. Please try again.",
				User: chat.ID,
			})
		}
		if isValid, errMsg := isValidAccountNumber(accountNumber); !isValid {
			text := fmt.Sprintf("***%s\n\nEnter your %s account number***", errMsg, bankData.Name)
			return Telegram.SendUserMessage(TelegramMessage{Text: text, User: chat.ID})
		}
		accountDetails, err := withdrawalService.ValidateBankAccount(accountNumber, bankCode)
		if err != nil {
			log.Error("error fetching banks", zap.Error(err))
			return Telegram.SendUserMessage(TelegramMessage{
				Text: "Failed to load banks. Please try again.",
				User: chat.ID,
			})
		}
		_ = database.DeleteRedisKey(fmt.Sprintf(common.RedisSetBankAccountNumberKey, chat.ID))
		var m strings.Builder
		m.WriteString(fmt.Sprintf(
			"📜 *Withdrawal Details*\n"+
				"  ━━━━━━━━━━━━━━  \n"+
				"🔖 *Amount:* %v\n"+
				"🔖 *Account Name:* %s\n"+
				"🆔 *Account Number:* `%s`\n"+
				"💰 *Bank:* %s\n"+
				"  ━━━━━━━━━━━━━━  \n",
			fmt.Sprintf(
				"**₦%s**",
				humanize.Commaf(withdrawalAmt),
			),
			accountDetails.AccountNumber,
			accountDetails.AccountName,
			bankData.Name,
		))
		buttons := [][]tgApi.InlineKeyboardButton{
			{
				{Text: "Confirm", CallbackData: helpers.StrPtr("confirm_withdrawal")},
			},
			{
				{Text: "Cancel", CallbackData: helpers.StrPtr("cancel_withdrawal")},
			},
		}
		replyMarkup := tgApi.InlineKeyboardMarkup{InlineKeyboard: buttons}

		return Telegram.SendUserMessage(TelegramMessage{Text: m.String(), User: chat.ID, ParseMode: "markdown", ReplyMarkup: &replyMarkup})
	}

	if state, _ := database.GetRedisKey(fmt.Sprintf(common.RedisConfirmWithdrawalPasswordKey, chat.ID)); state == "true" {
		user, err := telegramRepo.FindUserByTelegramID(int(telegramId))
		if err != nil {
			log.Error("error fetching user by telegram id", zap.Error(err))
			return sendErrorMessage(message.Chat.ID)
		}
		chatId := chat.ID

		passwordMatch, err := authService.ConfirmPassword(user.ID.String(), text)
		if err != nil {
			log.Error("error validating password", zap.Error(err))
			return sendErrorMessage(message.Chat.ID)
		}
		if !passwordMatch {
			buttons := [][]tgApi.InlineKeyboardButton{
				{
					{Text: "Retry", CallbackData: helpers.StrPtr("confirm_withdrawal")},
				},
				{
					{Text: "Cancel", CallbackData: helpers.StrPtr("cancel_withdrawal")},
				},
			}
			replyMarkup := tgApi.InlineKeyboardMarkup{InlineKeyboard: buttons}

			return Telegram.SendUserMessage(TelegramMessage{
				Text:        "🚫 *Invalid Password!* 🚫\n\n🔹 Please double-check and try again.",
				User:        chat.ID,
				ParseMode:   "Markdown",
				ReplyMarkup: &replyMarkup,
			})
		}

		redisKey := fmt.Sprintf(common.RedisSelectedBankKey, chatId)
		bankCode, err := database.GetRedisKey(redisKey)
		if err != nil {
			log.Error("error fetching from redis", zap.Error(err))
			return sendErrorMessage(chatId)
		}
		amountRedisKey := fmt.Sprintf(common.RedisWithdrawalAmountSetupKey, chatId)
		withdrawalAmtStr, err := database.GetRedisKey(amountRedisKey)
		if err != nil {
			log.Error("error fetching from redis", zap.Error(err))
			return sendErrorMessage(chatId)
		}
		withdrawalAmt, err := strconv.ParseFloat(withdrawalAmtStr, 64)
		if err != nil {
			log.Error("error fetching from redis", zap.Error(err))
			return sendErrorMessage(chatId)
		}
		redisKey = fmt.Sprintf(common.RedisBankAccountNumberKey, chatId)
		accountNumber, err := database.GetRedisKey(redisKey)
		if err != nil {
			log.Error("error fetching from redis", zap.Error(err))
			return sendErrorMessage(chatId)
		}
		if err := withdrawalService.InitiateTransfer(accountNumber, bankCode, user.ID.String(), withdrawalAmt); err != nil {
			log.Error("error initiating withdrawal", zap.Error(err))
			return sendErrorMessage(message.Chat.ID)
		}
		return Telegram.SendUserMessage(TelegramMessage{
			Text:      "✅ *Withdrawal in Progress!* ✅\n\n📩 We'll notify you shortly once the transaction is processed.",
			User:      chatId,
			ParseMode: "Markdown",
		})

	}

	return nil
}

func handleCallback(callbackQuery *tgApi.CallbackQuery) error {
	var (
		data       = callbackQuery.Data
		telegramId = callbackQuery.From.ID
	)

	if strings.HasPrefix(data, "generate_address:") {
		if err := Telegram.SendLoader(callbackQuery.Message.Chat.ID); err != nil {
			log.Error("error sending loader", zap.Error(err))
			return sendErrorMessage(callbackQuery.Message.Chat.ID)
		}
		assetID := strings.TrimPrefix(data, "generate_address:")

		redisKey := fmt.Sprintf(common.RedisAssetSelectionKey, callbackQuery.From.ID)
		err := database.SetRedisKey(redisKey, assetID, 300)
		if err != nil {
			log.Error("error setting redis key for asset selection", zap.Error(err))
			text := "Failed to process your selection. Please try again."
			return Telegram.SendCallbackResponse(common.TelegramCallbackResponse{
				CallbackQueryID: callbackQuery.ID,
				Text:            text,
				ShowAlert:       true,
			})
		}

		asset, err := assetRepo.FindAssetByID(assetID)
		if err != nil {
			log.Error("error fetching asset data", zap.Error(err))
			text := "Failed to process your selection. Please try again."
			return Telegram.SendCallbackResponse(common.TelegramCallbackResponse{
				CallbackQueryID: callbackQuery.ID,
				Text:            text,
				ShowAlert:       true,
			})
		}

		text := func() string {
			s := fmt.Sprintf("You selected  `%s`. Confirm to generate the address.", strings.ToUpper(asset.Symbol))
			if asset.Standard != "" {
				s = fmt.Sprintf("You selected  `%s (%s)`. Confirm to generate the address.", asset.Symbol, strings.ToUpper(asset.Standard))
			}
			return s
		}()

		confirmButtons := [][]tgApi.InlineKeyboardButton{
			{
				{Text: "Confirm", CallbackData: helpers.StrPtr("confirm_generate")},
			},
			{
				{Text: "Cancel", CallbackData: helpers.StrPtr("cancel_generate")},
			},
		}

		replyMarkup := tgApi.InlineKeyboardMarkup{InlineKeyboard: confirmButtons}

		err = Telegram.EditMessage(TelegramMessageEdit{
			ChatID:      callbackQuery.Message.Chat.ID,
			MessageID:   callbackQuery.Message.MessageID,
			NewText:     text,
			ReplyMarkup: &replyMarkup,
			ParseMode:   "markdown",
		})
		if err != nil {
			log.Error("error editing message for confirm/cancel", zap.Error(err))
		}

		err = database.SetRedisKey(fmt.Sprintf(common.RedisAssetSelectionKey, callbackQuery.Message.Chat.ID), assetID, 0)
		if err != nil {
			log.Error("error setting redis key", zap.Error(err))

		}
		return Telegram.SendCallbackResponse(common.TelegramCallbackResponse{
			CallbackQueryID: callbackQuery.ID,
		})
	}

	if data == "confirm_generate" {
		if err := Telegram.SendLoader(callbackQuery.Message.Chat.ID); err != nil {
			log.Error("error sending loader", zap.Error(err))
			return sendErrorMessage(callbackQuery.Message.Chat.ID)
		}
		redisKey := fmt.Sprintf(common.RedisAssetSelectionKey, callbackQuery.From.ID)
		assetID, err := database.GetRedisKey(redisKey)
		if err != nil || assetID == "" {
			log.Error("error fetching selected asset from redis", zap.Error(err))
			text := fmt.Sprintf("No asset selected or session expired. Please send command again.")
			return Telegram.SendCallbackResponse(common.TelegramCallbackResponse{
				CallbackQueryID: callbackQuery.ID,
				Text:            text,
				ShowAlert:       true,
			})
		}

		asset, err := assetRepo.FindAssetByID(assetID)
		if err != nil {
			log.Error("error fetching asset data", zap.Error(err))
			text := "Failed to process your selection. Please try again."
			return Telegram.SendCallbackResponse(common.TelegramCallbackResponse{
				CallbackQueryID: callbackQuery.ID,
				Text:            text,
				ShowAlert:       true,
			})
		}

		user, err := telegramRepo.FindUserByTelegramID(int(telegramId)) // user will always exist from the updateRecord middleware
		if err != nil {
			log.Error("error fetching user by telegram id", zap.Error(err))
			text := "Failed to process your selection. Please try again."
			return Telegram.SendCallbackResponse(common.TelegramCallbackResponse{
				CallbackQueryID: callbackQuery.ID,
				Text:            text,
				ShowAlert:       true,
			})
		}

		addressData, err := addressService.GetUserAddress(user, assetID)
		if err != nil {
			log.Error("error getting user address", zap.Error(err))
			text := "Failed to process your selection. Please try again."
			return Telegram.SendCallbackResponse(common.TelegramCallbackResponse{
				CallbackQueryID: callbackQuery.ID,
				Text:            text,
				ShowAlert:       true,
			})
		}

		_ = database.DeleteRedisKey(redisKey)

		err = Telegram.SendUserMessage(TelegramMessage{
			User:      callbackQuery.Message.Chat.ID,
			Text:      addressData.Address,
			ParseMode: "markdown",
		})
		if err != nil {
			log.Error("error editing message with generated address", zap.Error(err))
		}

		err = Telegram.SendCallbackResponse(common.TelegramCallbackResponse{
			CallbackQueryID: callbackQuery.ID,
		})
		if err != nil {
			log.Error("error editing message with generated address", zap.Error(err))
		}

		var m strings.Builder

		text := func() string {
			var sb strings.Builder

			sb.WriteString("_Copy the address above to send funds._\n\n")

			// sb.WriteString(fmt.Sprintf("📬 *Wallet Address:*\n`%s`\n\n", addressData.Address))

			sb.WriteString("ℹ️ *Instructions:*\n")
			sb.WriteString(fmt.Sprintf("• This is your personal `%s` wallet address.\n", strings.ToUpper(asset.Symbol)))
			sb.WriteString("• All coins received are converted to Naira using the displayed rate and credited to your Naira wallet.\n")
			sb.WriteString("• Any Naira balance is available for instant withdrawal to your bank account.\n")
			sb.WriteString(fmt.Sprintf("• *Only send %s (%s)* to this address.\n", strings.ToUpper(asset.Symbol), strings.ToUpper(asset.Standard)))

			sb.WriteString("\n")
			return sb.String()
		}()

		m.WriteString(text)

		footer, err := getFooter()
		if err != nil {
			log.Error("error getting user address", zap.Error(err))
			text := "Failed to process your selection. Please try again."
			return Telegram.SendCallbackResponse(common.TelegramCallbackResponse{
				CallbackQueryID: callbackQuery.ID,
				Text:            text,
				ShowAlert:       true,
			})
		}
		m.WriteString(footer)
		// err = Telegram.EditMessage(TelegramMessageEdit{
		// 	ChatID:    callbackQuery.Message.Chat.ID,
		// 	MessageID: callbackQuery.Message.MessageID,
		// 	NewText:   m.String(),
		// 	ParseMode: "markdown",
		// })
		// if err != nil {
		// 	log.Error("error editing message with generated address", zap.Error(err))
		// }
		//
		// return Telegram.SendCallbackResponse(common.TelegramCallbackResponse{
		// 	CallbackQueryID: callbackQuery.ID,
		// })
		err = Telegram.SendUserMessage(TelegramMessage{
			User:      callbackQuery.Message.Chat.ID,
			Text:      m.String(),
			ParseMode: "markdown",
		})
	}

	if data == "cancel_generate" {
		redisKey := fmt.Sprintf(common.RedisAssetSelectionKey, callbackQuery.From.ID)
		_ = database.DeleteRedisKey(redisKey)

		text := "Address generation canceled. You can use /generate to start again."
		err := Telegram.EditMessage(TelegramMessageEdit{
			ChatID:    callbackQuery.Message.Chat.ID,
			MessageID: callbackQuery.Message.MessageID,
			NewText:   text,
		})
		if err != nil {
			log.Error("error editing message for cancellation", zap.Error(err))
		}

		return Telegram.SendCallbackResponse(common.TelegramCallbackResponse{
			CallbackQueryID: callbackQuery.ID,
		})
	}

	if strings.HasPrefix(data, "transactions_page:") {
		if err := Telegram.SendLoader(callbackQuery.Message.Chat.ID); err != nil {
			log.Error("error sending loader", zap.Error(err))
			return sendErrorMessage(callbackQuery.Message.Chat.ID)
		}

		pageStr := strings.TrimPrefix(data, "transactions_page:")
		page, err := strconv.Atoi(pageStr)
		if err != nil || page < 1 {
			log.Error("invalid page number in callback", zap.Error(err))
			return Telegram.SendCallbackResponse(common.TelegramCallbackResponse{
				CallbackQueryID: callbackQuery.ID,
				Text:            "Invalid page number.",
			})
		}

		user, err := telegramRepo.FindUserByTelegramID(int(telegramId))
		if err != nil {
			log.Error("error fetching user by telegram id", zap.Error(err))
			return Telegram.SendCallbackResponse(common.TelegramCallbackResponse{
				CallbackQueryID: callbackQuery.ID,
				Text:            "Failed to process your selection. Please try again.",
				ShowAlert:       true,
			})
		}

		offset := (page - 1) * fetchTransactionLimit

		transactions, err := transactionService.FetchUserTransactions(user.ID.String(), fetchTransactionLimit, offset)
		if err != nil {
			log.Error("failed to fetch user transactions", zap.Error(err))
			return Telegram.SendCallbackResponse(common.TelegramCallbackResponse{
				CallbackQueryID: callbackQuery.ID,
				Text:            "Failed to fetch transactions. Please try again later.",
			})
		}

		totalTransactions, err := transactionService.FetchUserTransactionCount(user.ID.String())
		if err != nil {
			log.Error("failed to count user transactions", zap.Error(err))
			return Telegram.SendCallbackResponse(common.TelegramCallbackResponse{
				CallbackQueryID: callbackQuery.ID,
				Text:            "Failed to fetch transactions. Please try again later.",
			})
		}

		var message strings.Builder
		totalPages := (totalTransactions + fetchTransactionLimit - 1) / fetchTransactionLimit
		message.WriteString(fmt.Sprintf("*Your Transaction History (Page %d of %d):*\n\n", page, totalPages))
		for _, tx := range transactions {
			nairaBalance := humanize.Commaf(tx.Amount * tx.Rate)
			usdBalance := humanize.Commaf(tx.AmountUSD)
			message.WriteString(fmt.Sprintf(
				"📜 *Transaction Details*\n"+
					"  ━━━━━━━━━━━━━━  \n"+
					"🔖 *Type:* %s\n"+
					"🆔 *Transaction Reference:* `%s`\n"+
					"💰 *Currency:* %s\n"+
					"📊 *Amount:* `%.2f %s`\n"+
					"🔵 *Status:* %s\n"+
					"🔂 *Confirmations:* %d\n"+
					"📅 *Date:* %s\n"+
					"💹 *Rate:* `₦%.2f`\n"+
					"💵 *Amount in Naira:* `₦%s`\n"+
					"💲 *Amount in USD:* `$%s`\n"+
					"  ━━━━━━━━━━━━━━  \n",
				tx.Type,
				tx.Reference,
				func() string {
					s := fmt.Sprintf("%s", strings.ToUpper(tx.AssetSymbol))
					if tx.AssetStandard != "" {
						s = fmt.Sprintf("%s (%s)", strings.ToUpper(tx.AssetSymbol), strings.ToUpper(tx.AssetStandard))
					}
					return s
				}(),
				tx.Amount,
				tx.AssetSymbol,
				tx.Status,
				tx.Confirmations,
				tx.CreatedAt.Format("02 Jan 2006, 03:04 PM"),
				tx.Rate,
				nairaBalance,
				usdBalance,
			))
			message.WriteString("\n")
		}

		var buttons [][]tgApi.InlineKeyboardButton

		if page > 1 {
			prevPageButton := tgApi.NewInlineKeyboardButtonData(
				fmt.Sprintf("⬅️ Previous Page (%d)", page-1),
				fmt.Sprintf("transactions_page:%d", page-1),
			)
			buttons = append(buttons, []tgApi.InlineKeyboardButton{prevPageButton})
		}

		if totalTransactions > page*fetchTransactionLimit {
			nextPageButton := tgApi.NewInlineKeyboardButtonData(
				fmt.Sprintf("Next Page (%d) ➡️", page+1),
				fmt.Sprintf("transactions_page:%d", page+1),
			)
			if page > 1 {
				buttons[len(buttons)-1] = append(buttons[len(buttons)-1], nextPageButton)
			} else {
				buttons = append(buttons, []tgApi.InlineKeyboardButton{nextPageButton})
			}
		}

		replyMarkup := tgApi.NewInlineKeyboardMarkup(buttons...)
		err = Telegram.EditMessage(TelegramMessageEdit{
			ChatID:      callbackQuery.Message.Chat.ID,
			MessageID:   callbackQuery.Message.MessageID,
			NewText:     message.String(),
			ParseMode:   "Markdown",
			ReplyMarkup: &replyMarkup,
		})
		if err != nil {
			log.Error("error editing transactions message", zap.Error(err))
			return Telegram.SendCallbackResponse(common.TelegramCallbackResponse{
				CallbackQueryID: callbackQuery.ID,
				Text:            "Failed to update the message.",
			})
		}

		return Telegram.SendCallbackResponse(common.TelegramCallbackResponse{
			CallbackQueryID: callbackQuery.ID,
		})
	}

	if data == "withdraw_all" {
		chatId := callbackQuery.Message.Chat.ID
		err := database.DeleteRedisKey(fmt.Sprintf(common.RedisWithdrawSetupKey, chatId))
		if err != nil {
			log.Error("error deleting redis key", zap.Error(err))
		}
		user, err := telegramRepo.FindUserByTelegramID(int(telegramId))
		if err != nil {
			log.Error("error fetching user by telegram id", zap.Error(err))
			return sendErrorMessage(chatId)
		}
		wallet, err := walletService.GetUserWalletsData(user.ID.String())
		if err != nil {
			log.Error("failed to fetch user wallets", zap.Error(err))
			text := "Sorry, we couldn't retrieve your wallet balances at this time. Please try again later."
			return Telegram.SendUserMessage(TelegramMessage{Text: text, User: chatId})
		}
		err = database.SetRedisKey(fmt.Sprintf(common.RedisWithdrawalAmountSetupKey, chatId), fmt.Sprintf(`%v`, wallet.Balance), 0)
		if err != nil {
			log.Error("error setting redis key", zap.Error(err))
		}
		return getBanks(chatId)
	}

	if strings.HasPrefix(data, "banks_page:") {
		if err := Telegram.SendLoader(callbackQuery.Message.Chat.ID); err != nil {
			log.Error("error sending loader", zap.Error(err))
			return sendErrorMessage(callbackQuery.Message.Chat.ID)
		}
		pageStr := strings.TrimPrefix(data, "banks_page:")
		page, _ := strconv.Atoi(pageStr)
		paginatedBanks, totalPages, err := withdrawalService.GetBanks(page, fetchBankLimit)
		if err != nil {
			log.Error("error fetching banks", zap.Error(err))
			return Telegram.SendCallbackResponse(
				common.TelegramCallbackResponse{
					CallbackQueryID: callbackQuery.ID,
					Text:            "Failed to load banks. Please try again.",
					ShowAlert:       true,
				})
		}
		var buttons [][]tgApi.InlineKeyboardButton
		for _, bank := range paginatedBanks {
			buttons = append(buttons, []tgApi.InlineKeyboardButton{
				{
					Text:         bank.Name,
					CallbackData: helpers.StrPtr(fmt.Sprintf("select_bank:%s", bank.Code)),
				},
			})
		}
		if page < totalPages {
			buttons = append(buttons, []tgApi.InlineKeyboardButton{
				{
					Text:         "Next Page ➡️",
					CallbackData: helpers.StrPtr(fmt.Sprintf("banks_page:%d", page+1)),
				},
			})
		}
		buttons = append(buttons, []tgApi.InlineKeyboardButton{
			{
				Text:         "Search Bank 🔍",
				CallbackData: helpers.StrPtr("search_bank"),
			},
		})

		replyMarkup := tgApi.InlineKeyboardMarkup{InlineKeyboard: buttons}
		err = Telegram.EditMessage(TelegramMessageEdit{
			ChatID:      callbackQuery.Message.Chat.ID,
			MessageID:   callbackQuery.Message.MessageID,
			NewText:     "Please select your bank:",
			ParseMode:   "markdown",
			ReplyMarkup: &replyMarkup,
		})
		if err != nil {
			log.Error("error editing message with generated address", zap.Error(err))
		}

		return Telegram.SendCallbackResponse(common.TelegramCallbackResponse{
			CallbackQueryID: callbackQuery.ID,
		})
	}

	if strings.HasPrefix(data, "search_banks_page:") {
		if err := Telegram.SendLoader(callbackQuery.Message.Chat.ID); err != nil {
			log.Error("error sending loader", zap.Error(err))
			return sendErrorMessage(callbackQuery.Message.Chat.ID)
		}
		parts := strings.Split(data, ":")
		searchText := parts[2]
		pageStr := parts[1]
		page, _ := strconv.Atoi(pageStr)
		paginatedBanks, totalPages, err := withdrawalService.SearchBank(searchText, page, fetchBankLimit)
		if err != nil {
			log.Error("error fetching banks", zap.Error(err))
			return Telegram.SendCallbackResponse(
				common.TelegramCallbackResponse{
					CallbackQueryID: callbackQuery.ID,
					Text:            "Failed to load banks. Please try again.",
					ShowAlert:       true,
				})
		}
		var buttons [][]tgApi.InlineKeyboardButton
		for _, bank := range paginatedBanks {
			buttons = append(buttons, []tgApi.InlineKeyboardButton{
				{
					Text:         bank.Name,
					CallbackData: helpers.StrPtr(fmt.Sprintf("select_bank:%s", bank.Code)),
				},
			})
		}
		if page < totalPages {
			buttons = append(buttons, []tgApi.InlineKeyboardButton{
				{
					Text:         "Next Page ➡️",
					CallbackData: helpers.StrPtr(fmt.Sprintf("search_banks_page:%d:%s", page+1, searchText)),
				},
			})
		}
		buttons = append(buttons, []tgApi.InlineKeyboardButton{
			{
				Text:         "Search Bank 🔍",
				CallbackData: helpers.StrPtr("search_bank"),
			},
		})

		replyMarkup := tgApi.InlineKeyboardMarkup{InlineKeyboard: buttons}
		err = Telegram.EditMessage(TelegramMessageEdit{
			ChatID:      callbackQuery.Message.Chat.ID,
			MessageID:   callbackQuery.Message.MessageID,
			NewText:     "Please select your bank:",
			ParseMode:   "markdown",
			ReplyMarkup: &replyMarkup,
		})
		if err != nil {
			log.Error("error editing message with generated address", zap.Error(err))
		}

		return Telegram.SendCallbackResponse(common.TelegramCallbackResponse{
			CallbackQueryID: callbackQuery.ID,
		})
	}

	if strings.HasPrefix(data, "select_bank:") {
		bankCode := strings.TrimPrefix(data, "select_bank:")
		chatId := callbackQuery.Message.Chat.ID

		bankData, err := withdrawalService.GetBankByCode(bankCode)
		if err != nil {
			log.Error("error fetching banks", zap.Error(err))
			return Telegram.SendCallbackResponse(
				common.TelegramCallbackResponse{
					CallbackQueryID: callbackQuery.ID,
					Text:            "Failed to load banks. Please try again.",
					ShowAlert:       true,
				})
		}

		err = database.SetRedisKey(fmt.Sprintf(common.RedisSelectedBankKey, chatId), bankCode, 0)
		if err != nil {
			log.Error("error saving selected bank", zap.Error(err))
			return Telegram.SendCallbackResponse(
				common.TelegramCallbackResponse{
					CallbackQueryID: callbackQuery.ID,
					Text:            "Failed to load banks. Please try again.",
					ShowAlert:       true,
				})
		}

		err = database.SetRedisKey(fmt.Sprintf(common.RedisSetBankAccountNumberKey, chatId), "true", 0)
		if err != nil {
			log.Error("error setting redis key", zap.Error(err))
		}

		text := fmt.Sprintf("***Enter your %s account number***", bankData.Name)

		err = Telegram.SendUserMessage(TelegramMessage{
			Text:      text,
			User:      callbackQuery.Message.Chat.ID,
			ParseMode: "markdown",
		})
		if err != nil {
			log.Error("error editing message for confirm/cancel", zap.Error(err))
		}
		return Telegram.SendCallbackResponse(common.TelegramCallbackResponse{
			CallbackQueryID: callbackQuery.ID,
		})
	}

	if data == "search_bank" {
		chatId := callbackQuery.Message.Chat.ID
		err := database.SetRedisKey(fmt.Sprintf(common.RedisSearchBankKey, chatId), "true", 0)
		if err != nil {
			log.Error("error setting redis key", zap.Error(err))
		}

		return Telegram.SendUserMessage(TelegramMessage{
			Text:      "***Please type the name of the bank you want to search for:***",
			User:      chatId,
			ParseMode: "markdown",
		})
	}

	if data == "confirm_withdrawal" {
		chatId := callbackQuery.Message.Chat.ID
		err := database.SetRedisKey(fmt.Sprintf(common.RedisConfirmWithdrawalPasswordKey, chatId), "true", 0)
		if err != nil {
			log.Error("error setting redis key", zap.Error(err))
		}

		return Telegram.SendUserMessage(TelegramMessage{
			Text:      "*🔒 To keep your account secure, please enter your password to continue:*\n\n*⏳ Your session will expire in 60 seconds if not completed.*",
			User:      chatId,
			ParseMode: "markdown",
		})

	}

	if data == "cancel_withdrawal" {
		return Telegram.SendUserMessage(TelegramMessage{
			Text:      "***Withdrawal process terminated***",
			User:      callbackQuery.Message.Chat.ID,
			ParseMode: "markdown",
		})
	}

	return nil
}

func getFooter() (string, error) {
	rate, err := rateService.GetCurrentRate()
	if err != nil {
		log.Error("Error fetching exchange rate", zap.Error(err))
		return "", err
	}

	rateText := "\n\n💱 *Current Exchange Rate:* \n"
	rateText += fmt.Sprintf("📊 *₦%.2f / $*\n", rate.Rate)

	return rateText, nil
}

func getBanks(chatID int64) error {
	page := 1

	paginatedBanks, totalPages, err := withdrawalService.GetBanks(page, fetchBankLimit)
	if err != nil {
		log.Error("error fetching banks", zap.Error(err))
		return Telegram.SendUserMessage(TelegramMessage{
			Text: fmt.Sprintf("Failed to load banks. Please try again later."),
			User: chatID,
		})
	}
	var buttons [][]tgApi.InlineKeyboardButton
	for _, bank := range paginatedBanks {
		buttons = append(buttons, []tgApi.InlineKeyboardButton{
			{
				Text:         bank.Name,
				CallbackData: helpers.StrPtr(fmt.Sprintf("select_bank:%s", bank.Code)),
			},
		})
	}
	if page < totalPages {
		buttons = append(buttons, []tgApi.InlineKeyboardButton{
			{
				Text:         "Next Page ➡️",
				CallbackData: helpers.StrPtr(fmt.Sprintf("banks_page:%d", page+1)),
			},
		})
	}
	buttons = append(buttons, []tgApi.InlineKeyboardButton{
		{
			Text:         "Search Bank 🔍",
			CallbackData: helpers.StrPtr("search_bank"),
		},
	})
	replyMarkup := tgApi.InlineKeyboardMarkup{InlineKeyboard: buttons}
	return Telegram.SendUserMessage(TelegramMessage{
		Text:        "Please select your bank:",
		User:        chatID,
		ReplyMarkup: replyMarkup,
	})
}

func updateRecord(update tgApi.Update) {
	mu.Lock()
	defer mu.Unlock()
	var (
		user       *database.User
		username   string
		telegramId int
		chatId     int64
		text       string
		err        error
	)

	if update.Message != nil {
		chatId = update.Message.Chat.ID
		telegramId = int(update.Message.From.ID)
		username = func() string {
			if update.Message.From.UserName != "" {
				return update.Message.From.UserName
			} else if update.Message.From.FirstName != "" {
				return update.Message.From.FirstName
			}
			return update.Message.From.LastName
		}()
		t, _, _ := strings.Cut(update.Message.Text, " ")
		text = t
	} else if update.CallbackQuery != nil {
		chatId = update.CallbackQuery.Message.Chat.ID
		telegramId = int(update.CallbackQuery.Message.From.ID)
		username = update.CallbackQuery.Message.From.UserName
		username = func() string {
			if update.CallbackQuery.Message.From.UserName != "" {
				return update.CallbackQuery.Message.From.UserName
			} else if update.CallbackQuery.Message.From.FirstName != "" {
				return update.CallbackQuery.Message.From.FirstName
			}
			return update.CallbackQuery.Message.From.LastName
		}()
		t, _, _ := strings.Cut(update.CallbackQuery.Message.Text, " ")
		text = t
	}

	if username != "" {
		user, err = telegramRepo.Upsert(username, telegramId)
		if err != nil {
			log.Error("error finding or creating telegram user", zap.Error(err))
		}
	}

	var userId uuid.UUID
	if user != nil {
		userId = user.ID
		if chatId > 0 {
			metadata := common.TelegramChatMetadata{
				User:      username,
				ChatID:    chatId,
				UpdatedAt: time.Now(),
			}

			metadataJSON, err := json.Marshal(metadata)
			if err != nil {
				log.Error("failed to marshal metadata: %v", zap.Error(err))
				return
			}

			err = database.HSet(common.RedisActiveChatsKey, userId.String(), metadataJSON)
			if err != nil {
				log.Error("failed to store active chat: %v", zap.Error(err))
			}

		}
		if err := telegramCmdLogRepo.Create(&database.TelegramCommandLog{
			ID:          uuid.New(),
			UserID:      userId,
			UsageTime:   time.Now(),
			CommandName: text,
		}); err != nil {
			log.Error("error creating telegram cmd log", zap.Error(err))
		}
	}
}

func SendTelegramUserMessage(chatId int64, message string) error {
	return Telegram.SendUserMessage(TelegramMessage{Text: message, User: chatId, ParseMode: "markdown"})
}

func validateAmount(input string) (float64, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return 0, errors.New("input cannot be empty")
	}
	amount, err := strconv.ParseFloat(input, 64)
	if err != nil {
		return 0, errors.New("invalid amount, please enter a valid number")
	}
	if amount <= 0 {
		return 0, errors.New("amount must be greater than zero")
	}
	parts := strings.Split(input, ".")
	if len(parts) == 2 && len(parts[1]) > 2 {
		return 0, errors.New("amount cannot have more than 2 decimal places")
	}
	return amount, nil
}

func isValidAccountNumber(accountNumber string) (bool, string) {
	if _, err := strconv.Atoi(accountNumber); err != nil {
		return false, "Account number contains invalid characters. Only digits are allowed."
	}
	if len(accountNumber) != 10 {
		return false, fmt.Sprintf("Account number must be exactly 10 digits. Provided: %d digits.", len(accountNumber))
	}
	match, _ := regexp.MatchString("^[0-9]{10}$", accountNumber)
	if !match {
		return false, "Account number format is incorrect."
	}
	return true, "Account number is valid."
}
