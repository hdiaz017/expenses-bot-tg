package main

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// ========================================
// 🚀 ROUTER DE COMANDOS
// ========================================
// Esta función redirige el comando al handler correcto
func handleCommand(bot *tgbotapi.BotAPI, db *sql.DB, chatID int64, userID int64, text string) {
	switch {
	case text == "/start":
		handleStart(bot, chatID)
	case strings.HasPrefix(text, "/presupuestomes"):
		handleMonthlyBudgetCommand(bot, db, chatID, userID, text)
	case strings.HasPrefix(text, "/presupuesto"):
		handleBudgetCommand(bot, db, chatID, userID, text)
	case text == "/semana":
		handleWeekCommand(bot, db, chatID, userID)
	case text == "/mes":
		handleMonthStatusCommand(bot, db, chatID, userID)
	default:
		sendMessage(bot, chatID, "❌ Comando no reconocido.")
	}
}

// ========================================
// 📝 HANDLERS DE COMANDOS
// ========================================

func handleStart(bot *tgbotapi.BotAPI, chatID int64) {
	reply := `👋 **Hola, soy Juanito.**

Te ayudo a registrar tus gastos de forma rápida. Para que pueda entenderte, por favor sigue estas reglas:

📝 **Para registrar un gasto:**
Escribe el nombre y el monto **separados por un espacio**, sin signos de pesos ($) ni letras adicionales.
*Ejemplo:* ` + "`Torta 85`" + ` o ` + "`Bus 20`" + `
⚠️ *Solo un gasto por mensaje.*

💰 **Para definir tu presupuesto semanal y mensual:**
Usa el comando: /presupuesto 1500
Usa el comando: /presupuestomes 1500

📊 **Para ver cuánto te queda:**
Usa el comando: /semana
Usa el comando: /mes

*Si escribo algo que no entiendo, te avisaré para que lo intentes de nuevo.*`

	sendMessage(bot, chatID, reply)
}

func handleBudgetCommand(bot *tgbotapi.BotAPI, db *sql.DB, chatID int64, userID int64, text string) {
	parts := strings.Fields(text)
	if len(parts) != 2 {
		sendMessage(bot, chatID, "❌ Formato incorrecto.\n\nEjemplo:\n/presupuesto 1500")
		return
	}

	budget, err := strconv.ParseFloat(parts[1], 64)
	if err != nil || budget <= 0 {
		sendMessage(bot, chatID, "❌ El presupuesto debe ser un número mayor que 0.")
		return
	}

	if err := setWeeklyBudget(db, userID, budget); err != nil {
		log.Printf("Error guardando presupuesto: %v", err)
		sendMessage(bot, chatID, "❌ No pude guardar tu presupuesto.")
		return
	}

	sendMessage(bot, chatID, fmt.Sprintf("✅ Presupuesto semanal establecido: $%.2f", budget))
}

func handleMonthlyBudgetCommand(bot *tgbotapi.BotAPI, db *sql.DB, chatID int64, userID int64, text string) {
	parts := strings.Fields(text)
	if len(parts) < 2 {
		sendMessage(bot, chatID, "❌ Uso: `/presupuestomes 5000`")
		return
	}

	amount, err := strconv.ParseFloat(parts[1], 64)
	if err != nil || amount <= 0 {
		sendMessage(bot, chatID, "❌ El monto debe ser un número válido mayor a 0.")
		return
	}

	if err := setMonthlyBudget(db, userID, amount); err != nil {
		log.Printf("Error guardando presupuesto mensual: %v", err)
		sendMessage(bot, chatID, "❌ No pude guardar tu presupuesto mensual.")
		return
	}

	sendMessage(bot, chatID, fmt.Sprintf("✅ Presupuesto mensual actualizado a: $%.2f", amount))
}

func handleWeekCommand(bot *tgbotapi.BotAPI, db *sql.DB, chatID int64, userID int64) {
	spent, err := getWeeklySpent(db, userID)
	if err != nil {
		sendMessage(bot, chatID, "❌ Error al calcular gasto semanal.")
		return
	}

	budget, err := getWeeklyBudget(db, userID)
	if err != nil {
		sendMessage(bot, chatID, "❌ Error al obtener presupuesto.")
		return
	}

	if budget <= 0 {
		sendMessage(bot, chatID, fmt.Sprintf("📊 Esta semana has gastado: $%.2f\n\nNo tienes presupuesto establecido.\nUsa: /presupuesto 1500", spent))
		return
	}

	sendMessage(bot, chatID, fmt.Sprintf("📊 Esta semana\n\n💰 Gastado: $%.2f\n🎯 Presupuesto: $%.2f\n💵 Restante: $%.2f", spent, budget, budget-spent))
}

func handleMonthStatusCommand(bot *tgbotapi.BotAPI, db *sql.DB, chatID int64, userID int64) {
	spent, err := getMonthlySpent(db, userID)
	if err != nil {
		sendMessage(bot, chatID, "❌ Error al calcular gasto mensual.")
		return
	}

	budget, err := getMonthlyBudget(db, userID)
	if err != nil {
		sendMessage(bot, chatID, "❌ Error al obtener presupuesto mensual.")
		return
	}

	if budget <= 0 {
		sendMessage(bot, chatID, fmt.Sprintf("📅 **Estado Mensual**\n\nGastado: $%.2f\n\nNo tienes presupuesto mensual definido.\nUsa: `/presupuestomes 5000`", spent))
		return
	}

	sendMessage(bot, chatID, fmt.Sprintf("📅 **Estado Mensual**\n\n💰 Gastado: $%.2f\n🎯 Presupuesto: $%.2f\n💵 Restante: $%.2f", spent, budget, budget-spent))
}

// ========================================
// 🛒 REGISTRO DE GASTOS (Lógica Completa)
// ========================================

func handleExpenseRegistration(bot *tgbotapi.BotAPI, db *sql.DB, chatID int64, userID int64, text string) {
	// 1. Parsear el texto
	concept, amount, err := parseExpense(text)
	if err != nil {
		sendMessage(bot, chatID, "❌ "+err.Error())
		return
	}

	// 2. Guardar en DB
	if err := saveExpense(db, userID, concept, amount); err != nil {
		log.Printf("Error guardando gasto: %v", err)
		sendMessage(bot, chatID, "❌ No pude guardar el gasto.")
		return
	}

	// 3. Obtener datos para la respuesta automática (Semanal)
	weeklySpent, _ := getWeeklySpent(db, userID)
	weeklyBudget, _ := getWeeklyBudget(db, userID)

	// 4. Construir respuesta
	if weeklyBudget > 0 {
		reply := fmt.Sprintf(
			"✅ Registrado\n\n%s — $%.2f\n\n📊 Esta semana\nGastado: $%.2f\nPresupuesto: $%.2f\nRestante: $%.2f",
			concept, amount, weeklySpent, weeklyBudget, weeklyBudget-weeklySpent,
		)
		sendMessage(bot, chatID, reply)
	} else {
		reply := fmt.Sprintf(
			"✅ Registrado\n\n%s — $%.2f\n\n📊 Esta semana has gastado: $%.2f\n\nPuedes establecer tu presupuesto con:\n/presupuesto 1500",
			concept, amount, weeklySpent,
		)
		sendMessage(bot, chatID, reply)
	}
}

// ========================================
// ✉️ UTILIDADES
// ========================================

func sendMessage(bot *tgbotapi.BotAPI, chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown" // Para que funcionen las negritas y el código
	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error enviando mensaje: %v", err)
	}
}
