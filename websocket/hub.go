package websocket

import (
	"log"
	"sync"

	"github.com/gorilla/websocket"
)

type Client struct {
	ID     string
	UserID string
	Conn   *websocket.Conn
	send   chan Message
}

type Message struct {
	ID         string `json:"id,omitempty"`
	SenderID   string `json:"senderId"`
	ReceiverID string `json:"receiverId"`
	Content    string `json:"content"`
	Timestamp  int64  `json:"timestamp"`
}

type Hub struct {
	clients    map[*Client]bool
	broadcast  chan Message
	register   chan *Client
	unregister chan *Client
	users      map[string]*Client
	mutex      sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		broadcast:  make(chan Message),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
		users:      make(map[string]*Client),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mutex.Lock()
			h.clients[client] = true
			h.users[client.UserID] = client
			h.mutex.Unlock()
			log.Printf("Client registered: %s", client.UserID)

		case client := <-h.unregister:
			h.mutex.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				delete(h.users, client.UserID)
				close(client.send)
				log.Printf("Client unregistered: %s", client.UserID)
			}
			h.mutex.Unlock()

		case message := <-h.broadcast:
			h.mutex.RLock()
			if recipient, ok := h.users[message.ReceiverID]; ok {
				log.Printf("Broadcasting message from %s to %s", message.SenderID, message.ReceiverID)
				select {
				case recipient.send <- message:
				default:
					log.Printf("Recipient send buffer full, closing connection for user %s", recipient.UserID)
					close(recipient.send)
					delete(h.clients, recipient)
					delete(h.users, recipient.UserID)
				}
			} else {
				log.Printf("No active websocket recipient for user %s; message saved to DB only", message.ReceiverID)
			}
			h.mutex.RUnlock()
		}
	}
}

func (h *Hub) BroadcastMessage(msg Message) {
	select {
	case h.broadcast <- msg:
	default:
		log.Printf("Dropping broadcast message to receiver %s: hub busy", msg.ReceiverID)
	}
}

func (h *Hub) GetUserConnections() map[string]bool {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	onlineUsers := make(map[string]bool)
	for userID := range h.users {
		onlineUsers[userID] = true
	}
	return onlineUsers
}
