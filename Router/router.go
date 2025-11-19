package router

import (
	controller "BookingPlatfrom/Controller"
	"BookingPlatfrom/websocket"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Router struct {
	*mux.Router
	hub            *websocket.Hub
	chatController *controller.ChatController
}

func NewRouter(hub *websocket.Hub, chatController *controller.ChatController) *Router {
	r := &Router{
		Router:         mux.NewRouter(),
		hub:            hub,
		chatController: chatController,
	}
	r.setupRoutes()
	return r
}

func (r *Router) setupRoutes() {
	// Existing routes
	r.HandleFunc("/registration", controller.UserRegistration).Methods("POST")
	r.HandleFunc("/login", controller.UserLogin).Methods("POST")
	r.HandleFunc("/provider", controller.ProviderRegistration).Methods("POST")
	r.HandleFunc("/providers", controller.GetProviders).Methods("GET")
	r.HandleFunc("/providers/{providerId}/services", controller.GetServicesByProvider).Methods("GET")
	r.HandleFunc("/providers/{providerId}/schedule", controller.GetProviderSchedule).Methods("GET")
	r.HandleFunc("/service", controller.ServiceRegistration).Methods("POST")
	r.HandleFunc("/booking", controller.ServiceBooking).Methods("POST")
	r.HandleFunc("/bookings/me", controller.GetMyBookings).Methods("GET")
	r.HandleFunc("/providers/me/bookings", controller.GetMyProviderBookings).Methods("GET")
	r.HandleFunc("/bookings/{bookingId}/status", controller.UpdateBookingStatus).Methods("PUT")
	r.HandleFunc("/updateschedule", controller.UpdateProviderSchedule).Methods("PUT")
	r.HandleFunc("/webhooks/notifications", controller.HandleNotificationWebhook).Methods("POST")

	// Chat routes
	r.HandleFunc("/api/chats", r.GetChatHistory).Methods("GET")
	r.HandleFunc("/api/chats/recent", r.GetRecentChats).Methods("GET")
	r.HandleFunc("/api/messages", r.GetMessagesBetweenUsers).Methods("GET")
	r.HandleFunc("/api/messages", r.SendMessage).Methods("POST")

	// User lookup (returns basic public profile for user id)
	r.HandleFunc("/api/users/{id}", r.GetUser).Methods("GET")

	// WebSocket endpoint
	r.HandleFunc("/ws", r.handleWebSocket)

	// Debug endpoints
	r.HandleFunc("/api/debug/online-users", r.DebugOnlineUsers).Methods("GET")

	// Google OAuth routes
	r.HandleFunc("/auth/google/login", controller.GoogleLogin).Methods("GET")
	r.HandleFunc("/auth/google/callback", controller.GoogleCallback).Methods("GET")
}

// DebugOnlineUsers returns a JSON map of currently online users
func (r *Router) DebugOnlineUsers(w http.ResponseWriter, req *http.Request) {
	users := r.hub.GetUserConnections()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

// GetChatHistory handles GET /api/chats?userId=xxx&otherUserId=yyy
func (r *Router) GetChatHistory(w http.ResponseWriter, req *http.Request) {
	userID := req.URL.Query().Get("userId")
	otherUserID := req.URL.Query().Get("otherUserId")

	if userID == "" || otherUserID == "" {
		http.Error(w, "userId and otherUserId are required", http.StatusBadRequest)
		return
	}

	// Call the chat controller to get the chat history
	r.chatController.GetChatHistory(w, req)

	// Return the messages as JSON
	w.Header().Set("Content-Type", "application/json")
}

// GetRecentChats handles GET /api/chats/recent?userId=xxx
func (r *Router) GetRecentChats(w http.ResponseWriter, req *http.Request) {
	userID := req.URL.Query().Get("userId")
	if userID == "" {
		http.Error(w, "userId is required", http.StatusBadRequest)
		return
	}

	// Call the chat controller to get recent chats
	recentChats, err := r.chatController.GetRecentChats(userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return the recent chats as JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(recentChats)
}

// handleWebSocket handles WebSocket connections
func (r *Router) handleWebSocket(w http.ResponseWriter, req *http.Request) {
	userID := req.URL.Query().Get("userId")
	if userID == "" {
		http.Error(w, "userId is required", http.StatusBadRequest)
		return
	}
	websocket.ServeWs(r.hub, w, req, userID)
}

// GetMessagesBetweenUsers handles GET /api/messages?user1=:id1&user2=:id2
func (r *Router) GetMessagesBetweenUsers(w http.ResponseWriter, req *http.Request) {
	// Get query parameters
	user1ID := req.URL.Query().Get("user1")
	user2ID := req.URL.Query().Get("user2")

	if user1ID == "" || user2ID == "" {
		http.Error(w, "Both user1 and user2 parameters are required", http.StatusBadRequest)
		return
	}

	// Call the chat controller to get messages between users
	req.URL.RawQuery = "senderId=" + user1ID + "&receiverId=" + user2ID
	r.chatController.GetChatHistory(w, req)
}

// SendMessage handles POST /api/messages
func (r *Router) SendMessage(w http.ResponseWriter, req *http.Request) {
	// Parse request body
	var message controller.ChatMessage
	if err := json.NewDecoder(req.Body).Decode(&message); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if message.SenderID == "" || message.ReceiverID == "" || message.Content == "" {
		http.Error(w, "senderId, receiverId, and content are required", http.StatusBadRequest)
		return
	}

	// Save the message to the database
	if err := r.chatController.SaveMessage(message); err != nil {
		http.Error(w, "Failed to save message: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Broadcast the saved message to any connected websocket recipient for real-time delivery
	wsMsg := websocket.Message{
		SenderID:   message.SenderID,
		ReceiverID: message.ReceiverID,
		Content:    message.Content,
		Timestamp:  time.Now().Unix(),
	}

	// Send in a goroutine to avoid blocking the HTTP response
	// enqueue broadcast without blocking
	go r.hub.BroadcastMessage(wsMsg)

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Message sent successfully",
	})
}

// GetUser returns a public profile for the given user id
func (r *Router) GetUser(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	id := vars["id"]
	if id == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}

	// Convert to ObjectID
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	ctx := req.Context()
	var user map[string]interface{}
	if err := controller.UserCollection.FindOne(ctx, bson.M{"_id": oid}).Decode(&user); err != nil {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}
	fmt.Println(user)
	// Return only public fields
	result := map[string]interface{}{
		"_id":       id,
		"firstName": user["firstname"],
		"lastName":  user["lastname"],
		"email":     user["email"],
		"role":      user["role"],
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
