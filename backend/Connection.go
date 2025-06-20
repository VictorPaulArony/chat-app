package backend

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
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

// function to handle the websocket connecton for the users
func HandleConnections(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Error while upgrading net/http writer and request: ", err)
		return
	}
	defer ws.Close()

	// Read user Id from query parameters
	userID := 0
	query := r.URL.Query()
	if id := query.Get("user_id"); id != "" {
		fmt.Sscanf(id, "%d", &userID)
		return
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
	notifyUserStatusChange(userID, true)

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
			notifyUserStatusChange(userID, false)
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

func notifyUserStatusChange(userID int, online bool) {
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
