package controller

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log"
	"net/http"
	"os"
)

func HandleNotificationWebhook(w http.ResponseWriter, r *http.Request) {
	providerSignature := r.Header.Get("X-Provider-Signature")
	if providerSignature == "" {
		http.Error(w, "Missing signature header", http.StatusUnauthorized)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Cannot read request body", http.StatusInternalServerError)
		return
	}

	webhookSecret := os.Getenv("WEBHOOK_SECRET_KEY")

	mac := hmac.New(sha256.New, []byte(webhookSecret))
	mac.Write(body)
	expectedSignature := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(providerSignature), []byte(expectedSignature)) {
		log.Println("WARNING: Invalid webhook signature received.")
		http.Error(w, "Invalid signature", http.StatusForbidden)
		return
	}

	log.Printf("Received valid webhook. Body: %s", string(body))
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Webhook received successfully."))
}
