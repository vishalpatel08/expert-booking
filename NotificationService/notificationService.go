package main

import (
	"encoding/json"
	"log"
	"net/http"
)

type SendReminderRequest struct {
	BookingID string `json:"bookingId"`
	UserID    string `json:"userId"`
	UserEmail string `json:"userEmail"`
	Message   string `json:"message"`
}

func sendReminderHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	var req SendReminderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	log.Printf("Received SendReminder request for user %s with message: '%s'", req.UserID, req.Message)
	deliveryID := "id_12345"

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"deliveryId": deliveryID})
}

func main() {
	http.HandleFunc("/send-reminder", sendReminderHandler)
	log.Println("RESTful Notification Service started on port :50052")
	if err := http.ListenAndServe(":50052", nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
