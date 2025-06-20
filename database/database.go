package database

import (
	"database/sql"
	"log"
	"os"

	"golang.org/x/crypto/bcrypt"
)

var db *sql.DB

func InitDB() {
	var err error
	os.Remove("./db/chat.db")

	db, err = sql.Open("sqlite3", "./db/chat.db")
	if err != nil {
		log.Fatal(err)
	}

	// Create tables
	sqlStmt := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT UNIQUE NOT NULL,
		password TEXT NOT NULL,
		online BOOLEAN DEFAULT 0
	);
	CREATE TABLE IF NOT EXISTS messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		sender_id INTEGER NOT NULL,
		receiver_id INTEGER NOT NULL,
		content TEXT NOT NULL,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (sender_id) REFERENCES users(id),
		FOREIGN KEY (receiver_id) REFERENCES users(id)
	);
	`
	_, err = db.Exec(sqlStmt)
	if err != nil {
		log.Printf("%q: %s\n", err, sqlStmt)
		return
	}

	// Insert some test users if they don't exist
	testUsers := map[string]string{
		"alice":   "password123",
		"bob":     "password123",
		"charlie": "password123",
		"diana":   "password123",
	}

	for username, password := range testUsers {
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		_, err = db.Exec(`
			INSERT OR IGNORE INTO users (username, password) 
			VALUES (?, ?)`,
			username, hashedPassword)
		if err != nil {
			log.Printf("Error inserting user: %v\n", err)
		}
	}
}
