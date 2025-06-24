package backend

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/websocket"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type User struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Online   bool   `json:"online"`
}

type Message struct {
	ID         int       `json:"id"`
	SenderID   int       `json:"sender_id"`
	ReceiverID int       `json:"receiver_id"`
	Content    string    `json:"content"`
	Timestamp  time.Time `json:"timestamp"`
}

type Client struct {
	UserID int
	Conn   *websocket.Conn
}

var (
	clients   = make(map[int]*Client)
	broadcast = make(chan Message)
)

var db *sql.DB

func InitDB() {
	var err error
	os.Remove("./db/chat.db") // Remove if you want to persist data
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

func HandleConnections(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer ws.Close()

	// Read user ID from query params
	userID := 0
	query := r.URL.Query()
	if id := query.Get("user_id"); id != "" {
		fmt.Sscanf(id, "%d", &userID)
	}

	if userID == 0 {
		return
	}

	// Register client
	client := &Client{UserID: userID, Conn: ws}
	clients[userID] = client

	// Mark user as online
	_, err = db.Exec("UPDATE users SET online = 1 WHERE id = ?", userID)
	if err != nil {
		log.Printf("Error updating user status: %v\n", err)
	}

	// Notify all clients about the new online user
	NotifyUserStatusChange(userID, true)

	// Handle messages
	for {
		var msg Message
		err := ws.ReadJSON(&msg)
		if err != nil {
			log.Printf("error: %v", err)
			delete(clients, userID)

			// Mark user as offline
			_, err = db.Exec("UPDATE users SET online = 0 WHERE id = ?", userID)
			if err != nil {
				log.Printf("Error updating user status: %v\n", err)
			}

			// Notify all clients about the offline user
			NotifyUserStatusChange(userID, false)
			break
		}

		// Save message to database
		msg.Timestamp = time.Now()
		res, err := db.Exec(
			"INSERT INTO messages (sender_id, receiver_id, content, timestamp) VALUES (?, ?, ?, ?)",
			msg.SenderID, msg.ReceiverID, msg.Content, msg.Timestamp,
		)
		if err != nil {
			log.Printf("Error saving message: %v\n", err)
			continue
		}

		id, _ := res.LastInsertId()
		msg.ID = int(id)

		// Broadcast the message to the recipient
		broadcast <- msg
	}
}

func NotifyUserStatusChange(userID int, online bool) {
	// Get user details
	var username string
	err := db.QueryRow("SELECT username FROM users WHERE id = ?", userID).Scan(&username)
	if err != nil {
		log.Printf("Error getting user: %v\n", err)
		return
	}

	// Prepare status update message
	statusUpdate := map[string]interface{}{
		"type":     "user_status",
		"user_id":  userID,
		"username": username,
		"online":   online,
	}

	// Send to all connected clients
	for _, client := range clients {
		err := client.Conn.WriteJSON(statusUpdate)
		if err != nil {
			log.Printf("Error sending status update: %v\n", err)
		}
	}
}

func HandleMessages() {
	for {
		msg := <-broadcast

		// Send to sender (for their own message history)
		if sender, ok := clients[msg.SenderID]; ok {
			fullMsg := map[string]interface{}{
				"type":      "message",
				"message":   msg,
				"direction": "outgoing",
			}
			err := sender.Conn.WriteJSON(fullMsg)
			if err != nil {
				log.Printf("error: %v", err)
				sender.Conn.Close()
				delete(clients, msg.SenderID)
			}
		}

		// Send to receiver
		if receiver, ok := clients[msg.ReceiverID]; ok {
			fullMsg := map[string]interface{}{
				"type":      "message",
				"message":   msg,
				"direction": "incoming",
			}
			err := receiver.Conn.WriteJSON(fullMsg)
			if err != nil {
				log.Printf("error: %v", err)
				receiver.Conn.Close()
				delete(clients, msg.ReceiverID)
			}
		}
	}
}

// Update GetUsersHandler to handle database errors
func GetUsersHandler(w http.ResponseWriter, r *http.Request) {
	currentUserID := 0
	if id := r.URL.Query().Get("current_user_id"); id != "" {
		fmt.Sscanf(id, "%d", &currentUserID)
	}

	// Get all users
	rows, err := db.Query("SELECT id, username, online FROM users WHERE id != ?", currentUserID)
	if err != nil {
		log.Printf("Database error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		if err := rows.Scan(&user.ID, &user.Username, &user.Online); err != nil {
			log.Printf("Row scan error: %v", err)
			continue
		}
		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		log.Printf("Rows error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Rest of your sorting logic...
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(users); err != nil {
		log.Printf("JSON encode error: %v", err)
	}
}

func GetMessagesHandler(w http.ResponseWriter, r *http.Request) {
	currentUserID := 0
	otherUserID := 0

	if id := r.URL.Query().Get("current_user_id"); id != "" {
		fmt.Sscanf(id, "%d", &currentUserID)
	}
	if id := r.URL.Query().Get("other_user_id"); id != "" {
		fmt.Sscanf(id, "%d", &otherUserID)
	}

	if currentUserID == 0 || otherUserID == 0 {
		http.Error(w, "Invalid user IDs", http.StatusBadRequest)
		return
	}

	rows, err := db.Query(`
		SELECT id, sender_id, receiver_id, content, timestamp 
		FROM messages 
		WHERE (sender_id = ? AND receiver_id = ?) 
		OR (sender_id = ? AND receiver_id = ?)
		ORDER BY timestamp ASC
	`, currentUserID, otherUserID, otherUserID, currentUserID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var msg Message
		err := rows.Scan(&msg.ID, &msg.SenderID, &msg.ReceiverID, &msg.Content, &msg.Timestamp)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		messages = append(messages, msg)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(messages)
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var creds struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	err := json.NewDecoder(r.Body).Decode(&creds)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	var user User
	var hashedPassword string
	err = db.QueryRow("SELECT id, username, password FROM users WHERE username = ?", creds.Username).
		Scan(&user.ID, &user.Username, &hashedPassword)
	if err != nil {
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(creds.Password))
	if err != nil {
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	response := map[string]interface{}{
		"id":       user.ID,
		"username": user.Username,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
