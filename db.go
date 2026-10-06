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

func initDB() (*sql.DB, error) {
	db, err := sql.Open("sqlite", "./gastos.db")
	if err != nil {
		return nil, err
	}

	query := `
	CREATE TABLE IF NOT EXISTS expenses (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		concept TEXT NOT NULL,
		amount REAL NOT NULL,
		spent_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
	`

	_, err = db.Exec(query)
	if err != nil {
		return nil, err
	}

	return db, nil
}

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
