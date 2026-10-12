package main

import (
	"database/sql"
	"time"

	_ "modernc.org/sqlite"
)

type Expense struct {
	ID      int64
	UserID  int64
	Concept string
	Amount  float64
	SpentAt time.Time
}

type User struct {
	ID             int64
	TelegramUserID int64
	WeeklyBudget   float64
	MonthlyBudget  float64
}

func initDB() (*sql.DB, error) {
	db, err := sql.Open("sqlite", "./gastos.db")
	if err != nil {
		return nil, err
	}

	// Tabla de usuarios
	usersTable := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		telegram_user_id INTEGER UNIQUE NOT NULL,
		weekly_budget REAL NOT NULL DEFAULT 0,
		monthly_budget REAL DEFAULT 0
	);
	`

	_, err = db.Exec(usersTable)
	if err != nil {
		return nil, err
	}

	// Tabla de gastos
	expensesTable := `
	CREATE TABLE IF NOT EXISTS expenses (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		concept TEXT NOT NULL,
		amount REAL NOT NULL,
		spent_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id)
	);
	`

	_, err = db.Exec(expensesTable)
	if err != nil {
		return nil, err
	}

	return db, nil
}

// Crea el usuario si no existe y devuelve su ID interno.
func getOrCreateUser(db *sql.DB, telegramUserID int64) (int64, error) {
	// Intentamos obtener el usuario existente.
	var userID int64

	err := db.QueryRow(
		`SELECT id FROM users WHERE telegram_user_id = ?`,
		telegramUserID,
	).Scan(&userID)

	if err == nil {
		return userID, nil
	}

	// Si no existe, lo creamos.
	result, err := db.Exec(
		`INSERT INTO users (telegram_user_id) VALUES (?)`,
		telegramUserID,
	)
	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

// Establece el presupuesto semanal del usuario.
func setWeeklyBudget(
	db *sql.DB,
	userID int64,
	budget float64,
) error {
	_, err := db.Exec(
		`UPDATE users SET weekly_budget = ? WHERE id = ?`,
		budget,
		userID,
	)

	return err
}

// Obtiene el presupuesto semanal.
func getWeeklyBudget(
	db *sql.DB,
	userID int64,
) (float64, error) {
	var budget float64

	err := db.QueryRow(
		`SELECT weekly_budget FROM users WHERE id = ?`,
		userID,
	).Scan(&budget)

	return budget, err
}

// Guarda un gasto.
func saveExpense(
	db *sql.DB,
	userID int64,
	concept string,
	amount float64,
) error {
	query := `
	INSERT INTO expenses (user_id, concept, amount)
	VALUES (?, ?, ?)
	`

	_, err := db.Exec(
		query,
		userID,
		concept,
		amount,
	)

	return err
}

// Calcula cuánto ha gastado el usuario desde el inicio
// de la semana actual.
func getWeeklySpent(
	db *sql.DB,
	userID int64,
) (float64, error) {
	var spent float64

	query := `
	SELECT COALESCE(SUM(amount), 0)
	FROM expenses
	WHERE user_id = ?
	AND strftime('%Y-%W', spent_at) = strftime('%Y-%W', 'now')
	`

	err := db.QueryRow(
		query,
		userID,
	).Scan(&spent)

	return spent, err
}

// Obtener total gastado en el mes actual
func getMonthlySpent(db *sql.DB, userID int64) (float64, error) {
	var total float64
	query := `SELECT COALESCE(SUM(amount), 0) FROM expenses 
              WHERE user_id = ? AND strftime('%Y-%m', spent_at) = strftime('%Y-%m', 'now')`

	err := db.QueryRow(query, userID).Scan(&total)
	return total, err
}

// Obtener el presupuesto mensual del usuario
func getMonthlyBudget(db *sql.DB, userID int64) (float64, error) {
	var budget float64
	query := `SELECT monthly_budget FROM users WHERE id = ?`

	err := db.QueryRow(query, userID).Scan(&budget)
	return budget, err
}

// Guardar el presupuesto mensual
func setMonthlyBudget(db *sql.DB, userID int64, budget float64) error {
	_, err := db.Exec(`UPDATE users SET monthly_budget = ? WHERE id = ?`, budget, userID)
	if err != nil {
		return err
	}

	return nil
}
