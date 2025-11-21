package controller

import (
	auth "BookingPlatfrom/Auth"
	models "BookingPlatfrom/Models"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type AvailabilityUpdate struct {
	DayOfWeek string `json:"dayOfWeek"`
	StartTime string `json:"startTime" validate:"required"`
	EndTime   string `json:"endTime" validate:"required"`
}

func GetProviderSchedule(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	providerId := r.URL.Query().Get("providerId")
	if providerId == "" {
		parts := splitPath(r.URL.Path)
		if len(parts) >= 3 {
			providerId = parts[len(parts)-2]
		}
	}
	pid, err := primitive.ObjectIDFromHex(providerId)
	if err != nil {
		http.Error(w, "invalid providerId", http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var schedule models.ProviderSchedule
	if err := AvailabilityCollection.FindOne(ctx, bson.M{"providerId": pid}).Decode(&schedule); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			http.Error(w, "No schedule found for provider", http.StatusNotFound)
			return
		}
		log.Println("Error fetching provider schedule:", err)
		http.Error(w, "An internal server error occurred", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(schedule)
}

func splitPath(p string) []string {
	var parts []string
	start := 0
	for i := 0; i < len(p); i++ {
		if p[i] == '/' {
			if i > start {
				parts = append(parts, p[start:i])
			}
			start = i + 1
		}
	}
	if start < len(p) {
		parts = append(parts, p[start:])
	}
	return parts
}

func UpdateProviderSchedule(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	claims, err := auth.VerifyJwt(authHeader)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	if claims.Role != models.RoleProvider {
		http.Error(w, "Forbidden: Only providers can create a Availability", http.StatusForbidden)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Methods", "PUT")

	var payload AvailabilityUpdate
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := Validate.Struct(payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	filter := bson.M{"providerId": claims.UserId}
	ctx, cancle := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancle()
	var providerSchedule models.ProviderSchedule
	if err := AvailabilityCollection.FindOne(ctx, filter).Decode(&providerSchedule); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			http.Error(w, "No Provider found with given Id", http.StatusBadRequest)
			return
		}
		log.Println("Error finding Service:", err)
		http.Error(w, "An internal server error occurred", http.StatusInternalServerError)
		return
	}

	dayToUpdate := payload.DayOfWeek
	_, ok := providerSchedule.Week[dayToUpdate]
	if !ok {
		fmt.Printf("Error: Invalid day of week '%s' provided for update.", dayToUpdate)
		return
	}

	newTimeSlot := models.TimeSlot{
		IsAvailable: true,
		StartTime:   payload.StartTime,
		EndTime:     payload.EndTime,
	}
	providerSchedule.Week[dayToUpdate] = newTimeSlot
	update := bson.M{
		"$set": bson.M{"week": providerSchedule.Week},
	}

	filter = bson.M{"providerId": providerSchedule.ProviderID}
	_, err = AvailabilityCollection.UpdateOne(context.Background(), filter, update)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Availabilty marked successfully",
	})
}
