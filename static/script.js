document.addEventListener('DOMContentLoaded', function() {
    const app = document.getElementById('app');
    let currentUser = null;
    let selectedUser = null;
    let socket = null;

    // Event handler for window close
    window.addEventListener('beforeunload', function() {
        if (socket) {
            socket.close();
        }
    });

    // Render login screen
    function renderLogin() {
        app.innerHTML = `
            <div class="login-container">
                <div class="login-box">
                    <h2>Login</h2>
                    <form id="login-form">
                        <div class="form-group">
                            <label for="username">Username</label>
                            <input type="text" id="username" required>
                        </div>
                        <div class="form-group">
                            <label for="password">Password</label>
                            <input type="password" id="password" required>
                        </div>
                        <button type="submit">Login</button>
                    </form>
                    <div id="login-error" class="error-message"></div>
                </div>
            </div>
        `;

        const loginForm = document.getElementById('login-form');
        const loginError = document.getElementById('login-error');

        loginForm.addEventListener('submit', function(e) {
            e.preventDefault();

            const username = document.getElementById('username').value;
            const password = document.getElementById('password').value;

            fetch('/api/login', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({ username, password }),
            })
            .then(response => {
                if (!response.ok) {
                    throw new Error('Login failed');
                }
                return response.json();
            })
            .then(data => {
                currentUser = {
                    id: data.id,
                    username: data.username
                };
                renderChat();
                connectWebSocket();
                loadUserList();
            })
            .catch(error => {
                loginError.textContent = 'Invalid username or password';
                console.error('Login error:', error);
            });
        });
    }

    // Connect to WebSocket
    function connectWebSocket() {
        if (!currentUser) return;
        
        // Close existing connection if any
        if (socket) {
            socket.close();
        }
        
        socket = new WebSocket(`ws://${window.location.host}/ws?user_id=${currentUser.id}`);
        
        socket.onopen = function() {
            console.log('WebSocket connected');
            // Load user list immediately after connection
            loadUserList();
        };
        
        socket.onmessage = function(event) {
            try {
                const data = JSON.parse(event.data);
                
                if (data.type === 'message') {
                    if (selectedUser && 
                        (data.message.sender_id === selectedUser.id || data.message.receiver_id === selectedUser.id)) {
                        addMessageToChat(data.message, data.direction);
                    }
                } else if (data.type === 'user_status') {
                    loadUserList();
                }
            } catch (error) {
                console.error('Error processing WebSocket message:', error);
            }
        };
        
        socket.onclose = function() {
            console.log('WebSocket disconnected - attempting to reconnect...');
            // Attempt to reconnect after a delay
            setTimeout(connectWebSocket, 3000);
        };
        
        socket.onerror = function(error) {
            console.error('WebSocket error:', error);
        };
    }
    

    // Render chat interface
    function renderChat() {
        app.innerHTML = `
            <div class="chat-container">
                <div class="sidebar">
                    <div class="user-info">
                        <h3>Welcome, <span id="current-username">${currentUser.username}</span></h3>
                        <button id="logout-button">Logout</button>
                    </div>
                    <h3>Chat Users</h3>
                    <div id="user-list" class="user-list"></div>
                </div>
                <div class="chat-area">
                    <div id="chat-header" class="chat-header">
                        <h2>Select a user to chat with</h2>
                    </div>
                    <div id="messages" class="messages"></div>
                    <div class="message-input">
                        <input type="text" id="message-input" placeholder="Type your message..." disabled>
                        <button id="send-button" disabled>Send</button>
                    </div>
                </div>
            </div>
        `;

        const logoutButton = document.getElementById('logout-button');
        logoutButton.addEventListener('click', function() {
            if (socket) {
                socket.close();
            }
            currentUser = null;
            selectedUser = null;
            renderLogin();
        });

        const messageInput = document.getElementById('message-input');
        const sendButton = document.getElementById('send-button');

        messageInput.addEventListener('input', function() {
            sendButton.disabled = messageInput.value.trim() === '';
        });

        sendButton.addEventListener('click', sendMessage);
        messageInput.addEventListener('keypress', function(e) {
            if (e.key === 'Enter') {
                sendMessage();
            }
        });
    }

    function sendMessage() {
        const messageInput = document.getElementById('message-input');
        if (!selectedUser || !messageInput.value.trim()) return;

        const message = {
            sender_id: currentUser.id,
            receiver_id: selectedUser.id,
            content: messageInput.value.trim()
        };

        if (socket && socket.readyState === WebSocket.OPEN) {
            socket.send(JSON.stringify(message));
        }
        messageInput.value = '';
        document.getElementById('send-button').disabled = true;
    }

    // Load user list
    function loadUserList() {
        if (!currentUser) return;
        
        fetch(`/api/users?current_user_id=${currentUser.id}`)
            .then(response => {
                if (!response.ok) {
                    throw new Error(`HTTP error! status: ${response.status}`);
                }
                return response.json();
            })
            .then(users => {
                const userList = document.getElementById('user-list');
                if (!userList) return;
                
                userList.innerHTML = '';
                
                users.forEach(user => {
                    const userElement = document.createElement('div');
                    userElement.className = `user-item ${selectedUser && user.id === selectedUser.id ? 'active' : ''}`;
                    userElement.innerHTML = `
                        <span>${user.username}</span>
                        <span class="user-status ${user.online ? 'online' : 'offline'}"></span>
                    `;
                    
                    userElement.addEventListener('click', () => {
                        selectedUser = user;
                        document.querySelectorAll('.user-item').forEach(item => {
                            item.classList.remove('active');
                        });
                        userElement.classList.add('active');
                        
                        document.getElementById('chat-header').innerHTML = `
                            <h2>Chat with ${user.username}</h2>
                            <small>${user.online ? 'Online' : 'Offline'}</small>
                        `;
                        
                        document.getElementById('message-input').disabled = false;
                        loadMessages(user.id);
                    });
                    
                    userList.appendChild(userElement);
                });
            })
            .catch(error => {
                console.error('Error loading users:', error);
                // Show error to user
                const userList = document.getElementById('user-list');
                if (userList) {
                    userList.innerHTML = '<div class="error">Error loading users. Please try again.</div>';
                }
                // Retry after delay
                setTimeout(loadUserList, 3000);
            });
    }

    // Load messages with selected user
    function loadMessages(otherUserId) {
        if (!currentUser || !otherUserId) return;
        
        fetch(`/api/messages?current_user_id=${currentUser.id}&other_user_id=${otherUserId}`)
            .then(response => {
                if (!response.ok) {
                    throw new Error('Failed to load messages');
                }
                return response.json();
            })
            .then(messages => {
                const messagesContainer = document.getElementById('messages');
                messagesContainer.innerHTML = '';
                
                messages.forEach(message => {
                    const direction = message.sender_id === currentUser.id ? 'outgoing' : 'incoming';
                    addMessageToChat(message, direction);
                });
                
                messagesContainer.scrollTop = messagesContainer.scrollHeight;
            })
            .catch(error => {
                console.error('Error loading messages:', error);
            });
    }

    // Add message to chat
    function addMessageToChat(message, direction) {
        const messagesContainer = document.getElementById('messages');
        if (!messagesContainer) return;
        
        const messageElement = document.createElement('div');
        messageElement.className = `message ${direction}`;
        
        const date = new Date(message.timestamp);
        const timeString = date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
        const dateString = date.toLocaleDateString();
        
        messageElement.innerHTML = `
            <div class="message-info">${dateString} ${timeString}</div>
            <div class="message-content">${message.content}</div>
        `;
        
        messagesContainer.appendChild(messageElement);
        messagesContainer.scrollTop = messagesContainer.scrollHeight;
    }

    // Start with login screen
    renderLogin();
});