package backend

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"
)

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

func GetUsersHandler(w http.ResponseWriter, r *http.Request) {
	// Get current user ID from query params
	currentUserID := 0
	if id := r.URL.Query().Get("current_user_id"); id != "" {
		fmt.Sscanf(id, "%d", &currentUserID)
	}

	// Get all users
	rows, err := db.Query("SELECT id, username, online FROM users WHERE id != ?", currentUserID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		err := rows.Scan(&user.ID, &user.Username, &user.Online)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		users = append(users, user)
	}

	// Get last message for each user
	type LastMessage struct {
		UserID    int
		Timestamp time.Time
	}
	lastMessages := make(map[int]time.Time)

	query := `
	SELECT 
		CASE WHEN sender_id = ? THEN receiver_id ELSE sender_id END as other_user_id,
		MAX(timestamp) as last_timestamp
	FROM messages
	WHERE sender_id = ? OR receiver_id = ?
	GROUP BY other_user_id
	`
	rows, err = db.Query(query, currentUserID, currentUserID, currentUserID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var userID int
		var timestamp time.Time
		err := rows.Scan(&userID, &timestamp)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		lastMessages[userID] = timestamp
	}

	// Sort users by last message timestamp (newest first) or alphabetically
	sort.Slice(users, func(i, j int) bool {
		iTime, iHasMsg := lastMessages[users[i].ID]
		jTime, jHasMsg := lastMessages[users[j].ID]

		if iHasMsg && jHasMsg {
			return iTime.After(jTime)
		} else if iHasMsg {
			return true
		} else if jHasMsg {
			return false
		}
		return strings.ToLower(users[i].Username) < strings.ToLower(users[j].Username)
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
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
