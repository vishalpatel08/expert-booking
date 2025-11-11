package controller

import (
	auth "BookingPlatfrom/Auth"
	models "BookingPlatfrom/Models"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ProviderReg struct {
	Title          string `json:"title" validate:"required"`
	Domain         string `json:"domain" validate:"required"`
	Qualifications string `json:"qualifications"`
}

func ProviderRegistration(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		http.Error(w, "Missing authorization header", http.StatusUnauthorized)
		return
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	claims := &auth.Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(os.Getenv("JWT_SECRET_KEY")), nil
	})
	if err != nil || !token.Valid {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	filter := bson.M{"userId": claims.UserId}
	count, err := ProviderCollection.CountDocuments(context.Background(), filter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

	if count > 0 {
		http.Error(w, "Already Provider profile exist", http.StatusForbidden)
		return
	}

	if claims.Role != models.RoleProvider {
		http.Error(w, "Forbidden: Only providers can create a profile", http.StatusForbidden)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Allow-Control-Allow-Methods", "POST")

	var payload ProviderReg
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := Validate.Struct(payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	newProfile := models.ProviderProfile{
		UserId:         claims.UserId,
		Title:          payload.Title,
		Domain:         payload.Domain,
		Qualifications: payload.Qualifications,
		ServiceIds:     []primitive.ObjectID{},
	}

	_, err = ProviderCollection.InsertOne(context.Background(), newProfile)
	if err != nil {
		http.Error(w, "Failed to create provider", http.StatusInternalServerError)
		return
	}
	defaultWeek := setDefaultWeek()
	defaultSchedule := models.ProviderSchedule{
		ProviderID: newProfile.UserId,
		Week:       defaultWeek,
	}

	_, err = AvailabilityCollection.InsertOne(context.Background(), defaultSchedule)
	if err != nil {
		http.Error(w, "Failed to set default schedule", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{
		"message": "Provider registered successfully, Default Schedule is Set",
	})
}

func setDefaultWeek() map[string]models.TimeSlot {
	defaultWeek := make(map[string]models.TimeSlot)
	days := []string{"sunday", "monday", "tuesday", "wednesday", "thursday", "friday", "saturday"}
	for i, day := range days {
		if i >= 1 && i <= 5 {
			defaultWeek[day] = models.TimeSlot{
				IsAvailable: true,
				StartTime:   "09:00",
				EndTime:     "17:00",
			}
		} else {
			defaultWeek[day] = models.TimeSlot{
				IsAvailable: false,
				StartTime:   "",
				EndTime:     "",
			}
		}
	}
	return defaultWeek
}
