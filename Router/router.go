package router

import (
	controller "BookingPlatfrom/Controller"
	"BookingPlatfrom/websocket"
	"encoding/json"
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

	// User lookup
	r.HandleFunc("/api/users/{id}", r.GetUser).Methods("GET")

	// WebSocket endpoint
	r.HandleFunc("/ws", r.handleWebSocket)

	// Debug endpoints
	r.HandleFunc("/api/debug/online-users", r.DebugOnlineUsers).Methods("GET")

	// Google OAuth routes
	r.HandleFunc("/auth/google/login", controller.GoogleLogin).Methods("GET")
	r.HandleFunc("/auth/google/callback", controller.GoogleCallback).Methods("GET")
}

func (r *Router) DebugOnlineUsers(w http.ResponseWriter, req *http.Request) {
	users := r.hub.GetUserConnections()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

func (r *Router) GetChatHistory(w http.ResponseWriter, req *http.Request) {
	userID := req.URL.Query().Get("userId")
	otherUserID := req.URL.Query().Get("otherUserId")
	if userID == "" || otherUserID == "" {
		http.Error(w, "userId and otherUserId are required", http.StatusBadRequest)
		return
	}
	r.chatController.GetChatHistory(w, req)
	w.Header().Set("Content-Type", "application/json")
}

func (r *Router) GetRecentChats(w http.ResponseWriter, req *http.Request) {
	userID := req.URL.Query().Get("userId")
	if userID == "" {
		http.Error(w, "userId is required", http.StatusBadRequest)
		return
	}

	recentChats, err := r.chatController.GetRecentChats(userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(recentChats)
}

func (r *Router) handleWebSocket(w http.ResponseWriter, req *http.Request) {
	userID := req.URL.Query().Get("userId")
	if userID == "" {
		http.Error(w, "userId is required", http.StatusBadRequest)
		return
	}
	websocket.ServeWs(r.hub, w, req, userID)
}

func (r *Router) GetMessagesBetweenUsers(w http.ResponseWriter, req *http.Request) {
	user1ID := req.URL.Query().Get("user1")
	user2ID := req.URL.Query().Get("user2")

	if user1ID == "" || user2ID == "" {
		http.Error(w, "Both user1 and user2 parameters are required", http.StatusBadRequest)
		return
	}

	req.URL.RawQuery = "senderId=" + user1ID + "&receiverId=" + user2ID
	r.chatController.GetChatHistory(w, req)
}

func (r *Router) SendMessage(w http.ResponseWriter, req *http.Request) {
	var message controller.ChatMessage
	if err := json.NewDecoder(req.Body).Decode(&message); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if message.SenderID == "" || message.ReceiverID == "" || message.Content == "" {
		http.Error(w, "senderId, receiverId, and content are required", http.StatusBadRequest)
		return
	}

	if err := r.chatController.SaveMessage(message); err != nil {
		http.Error(w, "Failed to save message: "+err.Error(), http.StatusInternalServerError)
		return
	}

	wsMsg := websocket.Message{
		SenderID:   message.SenderID,
		ReceiverID: message.ReceiverID,
		Content:    message.Content,
		Timestamp:  time.Now().Unix(),
	}

	go r.hub.BroadcastMessage(wsMsg)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Message sent successfully",
	})
}

func (r *Router) GetUser(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	id := vars["id"]
	if id == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}

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
