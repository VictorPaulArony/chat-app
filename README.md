# Chat Application

A real-time chat application built with Go backend and modern JavaScript frontend. Features include:

- Real-time messaging using WebSocket
- User authentication and login
- Online/offline user status
- Message history
- Modern, responsive UI
- Error handling and user feedback

## Features

- Real-time chat between users
- User authentication with login
- Shows online/offline status of users
- Message history loading
- Modern, responsive UI
- Error handling and user feedback
- Automatic WebSocket reconnection
- Message timestamp display

## Tech Stack

- **Backend**: Go with Gorilla WebSocket
- **Frontend**: HTML5, CSS3, JavaScript
- **Database**: SQLite
- **WebSocket**: Real-time communication

## Getting Started

### Prerequisites

- Go 1.20 or higher
- SQLite3
- Modern web browser

### Installation

1. Clone the repository:
```bash
git clone https://github.com/VictorPaulArony/chat-app.git
cd chat-app
```

2. Install Go dependencies:
```bash
go mod tidy
```

3. Run the application:
```bash
go run main.go
```

4. Open your browser and navigate to `http://localhost:8080`

## Project Structure

```
chat-app/
├── backend/           # Backend Go code
│   └── Connection.go  # WebSocket and message handling
├── database/          # Database related code
├── static/            # Frontend assets
│   ├── script.js      # Main JavaScript code
│   ├── style.css      # CSS styles
│   └── index.html     # Main HTML file
└── main.go            # Main application entry point
```

## Usage

1. Open the application in your web browser
2. Click "Login" to access the chat interface
3. Enter your username and password
4. Select a user from the list to start chatting
5. Messages are sent in real-time using WebSocket
6. User online/offline status is automatically updated

## Contributing

1. Fork the repository
2. Create your feature branch
3. Commit your changes
4. Push to the branch
5. Create a Pull Request

## License

This project is licensed under the MIT License - see the LICENSE file for details

## Acknowledgments

- Thanks to the Gorilla WebSocket team for their excellent WebSocket implementation
- Thanks to all contributors and users who help improve this project