package main

import (
	"log"
	"os"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("No se encontró .env")
	}
	token := os.Getenv("TELEGRAM_BOT_TOKEN")

	if token == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN no está configurado")
	}

	// Inicializar DB
	db, err := initDB()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Inicializar Telegram
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Bot iniciado como @%s", bot.Self.UserName)

	// Long polling
	config := tgbotapi.NewUpdate(0)
	config.Timeout = 30

	updates := bot.GetUpdatesChan(config)

	for update := range updates {
		if update.Message == nil {
			continue
		}

		message := update.Message
		telegramUserID := message.From.ID
		text := strings.TrimSpace(message.Text)

		userID, err := getOrCreateUser(db, telegramUserID)
		if err != nil {
			sendMessage(bot, message.Chat.ID, "❌ Error con tu usuario.")
			continue
		}

		// Ruteo: Si empieza con /, va a handlers.handleCommand
		if strings.HasPrefix(text, "/") {
			handleCommand(bot, db, message.Chat.ID, userID, text)
			continue
		}

		// Si no es comando, intentamos registrar un gasto
		handleExpenseRegistration(bot, db, message.Chat.ID, userID, text)
	}
}
