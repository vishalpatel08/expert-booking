package controller

import (
	auth "BookingPlatfrom/Auth"
	models "BookingPlatfrom/Models"
	"context"
	"encoding/json"
	"net/http"
)

type ServiceReg struct {
	Title    string  `json:"title" validate:"required"`
	Duration int64   `json:"duration" validate:"required"`
	Price    float64 `json:"price" validate:"required"`
}

func ServiceRegistration(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	claims, err := auth.VerifyJwt(authHeader)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
	}
	if claims.Role != models.RoleProvider {
		http.Error(w, "Forbidden: Only providers can create a service", http.StatusForbidden)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Allow-Control-Allow-Methods", "POST")

	var payload ServiceReg
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := Validate.Struct(payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	newProfile := models.Service{
		ProviderId: claims.UserId,
		Title:      payload.Title,
		Duration:   payload.Duration,
		Price:      payload.Price,
	}

	_, err = ServiceCollection.InsertOne(context.Background(), newProfile)
	if err != nil {
		http.Error(w, "Failed to create service", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Service registered successfully",
	})
}
