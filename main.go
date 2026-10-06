package main

import (
	"fmt"
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

	// Configuración de long polling
	config := tgbotapi.NewUpdate(0)
	config.Timeout = 30

	updates := bot.GetUpdatesChan(config)

	for update := range updates {

		if update.Message == nil {
			continue
		}

		message := update.Message

		userID := message.From.ID
		text := strings.TrimSpace(message.Text)

		log.Printf(
			"Usuario %d: %s",
			userID,
			text,
		)

		// /start
		if text == "/start" {
			reply := "👋 Hola. Soy Juanito.\n\n" +
				"Envíame un gasto así:\n\n" +
				"Torta 85\n" +
				"Bus 20\n" +
				"Supermercado 450"

			sendMessage(bot, message.Chat.ID, reply)
			continue
		}

		// Ignorar comandos por ahora
		if strings.HasPrefix(text, "/") {
			sendMessage(
				bot,
				message.Chat.ID,
				"Comando no reconocido.",
			)
			continue
		}

		// Parsear gasto
		concept, amount, err := parseExpense(text)

		if err != nil {
			sendMessage(
				bot,
				message.Chat.ID,
				"❌ "+err.Error(),
			)
			continue
		}

		// Guardar en SQLite
		err = saveExpense(
			db,
			userID,
			concept,
			amount,
		)

		if err != nil {
			log.Printf(
				"Error guardando gasto: %v",
				err,
			)

			sendMessage(
				bot,
				message.Chat.ID,
				"❌ No pude guardar el gasto.",
			)

			continue
		}

		// Confirmar
		reply := fmt.Sprintf(
			"✅ Registrado\n\n%s — $%.2f",
			concept,
			amount,
		)

		sendMessage(
			bot,
			message.Chat.ID,
			reply,
		)
	}
}

func sendMessage(
	bot *tgbotapi.BotAPI,
	chatID int64,
	text string,
) {
	msg := tgbotapi.NewMessage(
		chatID,
		text,
	)

	_, err := bot.Send(msg)

	if err != nil {
		log.Printf(
			"Error enviando mensaje: %v",
			err,
		)
	}
}
