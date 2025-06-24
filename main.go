package main

import (
	"database/sql"
	"log"
	"net/http"
	"path/filepath"

	"chat-app/backend"

	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

func main() {
	backend.InitDB()
	// defer db.Close()

	// Start message broadcaster
	go backend.HandleMessages()

	// Serve static files
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/", fs)

	// WebSocket route
	http.HandleFunc("/ws", backend.HandleConnections)

	// API routes
	http.HandleFunc("/api/users", backend.GetUsersHandler)
	http.HandleFunc("/api/messages", backend.GetMessagesHandler)
	http.HandleFunc("/api/login", backend.LoginHandler)

	// Add favicon route to prevent 404 errors
	http.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		faviconPath := filepath.Join("static", "favicon.ico")
		http.ServeFile(w, r, faviconPath)
	})
	// http.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
	// 	http.ServeFile(w, r, "./static/favicon.ico")
	// })

	log.Println("Server starting on :8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
