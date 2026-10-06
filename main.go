package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
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

		log.Printf(
			"Usuario %d: %s",
			telegramUserID,
			text,
		)

		// Crear usuario si todavía no existe.
		userID, err := getOrCreateUser(
			db,
			telegramUserID,
		)

		if err != nil {
			log.Printf(
				"Error obteniendo usuario: %v",
				err,
			)

			sendMessage(
				bot,
				message.Chat.ID,
				"❌ Ha ocurrido un error con tu usuario.",
			)

			continue
		}

		// =========================
		// /start
		// =========================

		if text == "/start" {
			reply := `👋 Hola. Soy Jackito.

Puedo ayudarte a registrar tus gastos y controlar tu presupuesto semanal.

Ejemplos:

Torta 85
Bus 20
Supermercado 450

También puedes establecer tu presupuesto:

/presupuesto 1500

Y consultar tu situación semanal:

/semana`

			sendMessage(
				bot,
				message.Chat.ID,
				reply,
			)

			continue
		}

		// =========================
		// /presupuesto
		// =========================

		if strings.HasPrefix(text, "/presupuesto") {
			handleBudgetCommand(
				bot,
				db,
				message.Chat.ID,
				userID,
				text,
			)

			continue
		}

		// =========================
		// /semana
		// =========================

		if text == "/semana" {
			handleWeekCommand(
				bot,
				db,
				message.Chat.ID,
				userID,
			)

			continue
		}

		// =========================
		// Otros comandos
		// =========================

		if strings.HasPrefix(text, "/") {
			sendMessage(
				bot,
				message.Chat.ID,
				"❌ Comando no reconocido.",
			)

			continue
		}

		// =========================
		// Registrar gasto
		// =========================

		concept, amount, err := parseExpense(text)

		if err != nil {
			sendMessage(
				bot,
				message.Chat.ID,
				"❌ "+err.Error(),
			)

			continue
		}

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

		// Obtener información actualizada de la semana.
		weeklySpent, err := getWeeklySpent(
			db,
			userID,
		)

		if err != nil {
			log.Printf(
				"Error calculando gasto semanal: %v",
				err,
			)
		}

		weeklyBudget, err := getWeeklyBudget(
			db,
			userID,
		)

		if err != nil {
			log.Printf(
				"Error obteniendo presupuesto: %v",
				err,
			)
		}

		// =========================
		// Respuesta
		// =========================

		if weeklyBudget > 0 {
			remaining := weeklyBudget - weeklySpent

			reply := fmt.Sprintf(
				"✅ Registrado\n\n"+
					"%s — $%.2f\n\n"+
					"📊 Esta semana\n"+
					"Gastado: $%.2f\n"+
					"Presupuesto: $%.2f\n"+
					"Restante: $%.2f",
				concept,
				amount,
				weeklySpent,
				weeklyBudget,
				remaining,
			)

			sendMessage(
				bot,
				message.Chat.ID,
				reply,
			)
		} else {
			reply := fmt.Sprintf(
				"✅ Registrado\n\n"+
					"%s — $%.2f\n\n"+
					"📊 Esta semana has gastado: $%.2f\n\n"+
					"Puedes establecer tu presupuesto con:\n"+
					"/presupuesto 1500",
				concept,
				amount,
				weeklySpent,
			)

			sendMessage(
				bot,
				message.Chat.ID,
				reply,
			)
		}
	}
}

// ========================================
// /presupuesto
// ========================================

func handleBudgetCommand(
	bot *tgbotapi.BotAPI,
	db *sql.DB,
	chatID int64,
	userID int64,
	text string,
) {
	parts := strings.Fields(text)

	if len(parts) != 2 {
		sendMessage(
			bot,
			chatID,
			"❌ Formato incorrecto.\n\nEjemplo:\n/presupuesto 1500",
		)

		return
	}

	budget, err := strconv.ParseFloat(
		parts[1],
		64,
	)

	if err != nil || budget <= 0 {
		sendMessage(
			bot,
			chatID,
			"❌ El presupuesto debe ser un número mayor que 0.\n\nEjemplo:\n/presupuesto 1500",
		)

		return
	}

	err = setWeeklyBudget(
		db,
		userID,
		budget,
	)

	if err != nil {
		log.Printf(
			"Error guardando presupuesto: %v",
			err,
		)

		sendMessage(
			bot,
			chatID,
			"❌ No pude guardar tu presupuesto.",
		)

		return
	}

	sendMessage(
		bot,
		chatID,
		fmt.Sprintf(
			"✅ Presupuesto semanal establecido: $%.2f",
			budget,
		),
	)
}

// ========================================
// /semana
// ========================================

func handleWeekCommand(
	bot *tgbotapi.BotAPI,
	db *sql.DB,
	chatID int64,
	userID int64,
) {
	weeklySpent, err := getWeeklySpent(
		db,
		userID,
	)

	if err != nil {
		log.Printf(
			"Error calculando gasto semanal: %v",
			err,
		)

		sendMessage(
			bot,
			chatID,
			"❌ No pude calcular tu gasto semanal.",
		)

		return
	}

	weeklyBudget, err := getWeeklyBudget(
		db,
		userID,
	)

	if err != nil {
		log.Printf(
			"Error obteniendo presupuesto: %v",
			err,
		)

		sendMessage(
			bot,
			chatID,
			"❌ No pude obtener tu presupuesto.",
		)

		return
	}

	if weeklyBudget <= 0 {
		sendMessage(
			bot,
			chatID,
			fmt.Sprintf(
				"📊 Esta semana has gastado: $%.2f\n\n"+
					"Todavía no tienes un presupuesto establecido.\n\n"+
					"Puedes establecerlo con:\n"+
					"/presupuesto 1500",
				weeklySpent,
			),
		)

		return
	}

	remaining := weeklyBudget - weeklySpent

	reply := fmt.Sprintf(
		"📊 Esta semana\n\n"+
			"💰 Gastado: $%.2f\n"+
			"🎯 Presupuesto: $%.2f\n"+
			"💵 Restante: $%.2f",
		weeklySpent,
		weeklyBudget,
		remaining,
	)

	sendMessage(
		bot,
		chatID,
		reply,
	)
}

// ========================================
// Enviar mensaje
// ========================================

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
