document.addEventListener('DOMContentLoaded', function() {
    const loginContainer = document.getElementById('login-container');
    const chatContainer = document.getElementById('chat-container');
    const loginForm = document.getElementById('login-form');
    const loginError = document.getElementById('login-error');
    const currentUsername = document.getElementById('current-username');
    const logoutButton = document.getElementById('logout-button');
    const userList = document.getElementById('user-list');
    const chatHeader = document.getElementById('chat-header');
    const messagesContainer = document.getElementById('messages');
    const messageInput = document.getElementById('message-input');
    const sendButton = document.getElementById('send-button');
    
    let currentUser = null;
    let selectedUser = null;
    let socket = null;
    
    // Handle login
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
            
            // Update UI
            currentUsername.textContent = currentUser.username;
            loginContainer.style.display = 'none';
            chatContainer.style.display = 'flex';
            
            // Connect WebSocket
            connectWebSocket();
            
            // Load user list
            loadUserList();
        })
        .catch(error => {
            loginError.textContent = 'Invalid username or password';
            console.error('Login error:', error);
        });
    });
    
    // Handle logout
    logoutButton.addEventListener('click', function() {
        if (socket) {
            socket.close();
        }
        
        // Reset UI
        currentUser = null;
        selectedUser = null;
        loginContainer.style.display = 'flex';
        chatContainer.style.display = 'none';
        messagesContainer.innerHTML = '';
        chatHeader.innerHTML = '<h2>Select a user to chat with</h2>';
        document.getElementById('username').value = '';
        document.getElementById('password').value = '';
        loginError.textContent = '';
    });
    
    // Connect to WebSocket
    function connectWebSocket() {
        if (!currentUser) return;
        
        socket = new WebSocket(`ws://${window.location.host}/ws?user_id=${currentUser.id}`);
        
        socket.onopen = function() {
            console.log('WebSocket connected');
        };
        
        socket.onmessage = function(event) {
            const data = JSON.parse(event.data);
            
            if (data.type === 'message') {
                // Add message to chat if it's with the selected user
                if (selectedUser && 
                    (data.message.sender_id === selectedUser.id || data.message.receiver_id === selectedUser.id)) {
                    addMessageToChat(data.message, data.direction);
                }
                
                // Update user list to reflect new message order
                loadUserList();
            } else if (data.type === 'user_status') {
                // Update user online status in the list
                loadUserList();
            }
        };
        
        socket.onclose = function() {
            console.log('WebSocket disconnected');
        };
    }
    
    // Load user list
    function loadUserList() {
        if (!currentUser) return;
        
        fetch(`/api/users?current_user_id=${currentUser.id}`)
            .then(response => {
                if (!response.ok) {
                    throw new Error('Failed to load users');
                }
                return response.json();
            })
            .then(users => {
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
                        
                        // Highlight selected user
                        document.querySelectorAll('.user-item').forEach(item => {
                            item.classList.remove('active');
                        });
                        userElement.classList.add('active');
                        
                        // Update chat header
                        chatHeader.innerHTML = `
                            <h2>Chat with ${user.username}</h2>
                            <small>${user.online ? 'Online' : 'Offline'}</small>
                        `;
                        
                        // Enable message input
                        messageInput.disabled = false;
                        sendButton.disabled = false;
                        
                        // Load messages
                        loadMessages(user.id);
                    });
                    
                    userList.appendChild(userElement);
                });
            })
            .catch(error => {
                console.error('Error loading users:', error);
            });
    }
    
    // Load messages between current user and selected user
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
                messagesContainer.innerHTML = '';
                
                messages.forEach(message => {
                    const direction = message.sender_id === currentUser.id ? 'outgoing' : 'incoming';
                    addMessageToChat(message, direction);
                });
                
                // Scroll to bottom
                messagesContainer.scrollTop = messagesContainer.scrollHeight;
            })
            .catch(error => {
                console.error('Error loading messages:', error);
            });
    }
    
    // Add a message to the chat UI
    function addMessageToChat(message, direction) {
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
        
        // Scroll to bottom
        messagesContainer.scrollTop = messagesContainer.scrollHeight;
    }
    
    // Send message
    function sendMessage() {
        if (!currentUser || !selectedUser || !messageInput.value.trim()) return;
        
        const message = {
            sender_id: currentUser.id,
            receiver_id: selectedUser.id,
            content: messageInput.value.trim()
        };
        
        if (socket && socket.readyState === WebSocket.OPEN) {
            socket.send(JSON.stringify(message));
        }
        
        // Clear input
        messageInput.value = '';
    }
    
    // Send message on button click
    sendButton.addEventListener('click', sendMessage);
    
    // Send message on Enter key
    messageInput.addEventListener('keypress', function(e) {
        if (e.key === 'Enter') {
            sendMessage();
        }
    });
});