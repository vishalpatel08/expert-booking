package controller

import (
	auth "BookingPlatfrom/Auth"
	models "BookingPlatfrom/Models"
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type BookingReg struct {
	ServiceId primitive.ObjectID `json:"serviceId,omitempty" validate:"required"`
	StartTime time.Time          `json:"startTime,omitempty" validate:"required"`
}

func ServiceBooking(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	claims, err := auth.VerifyJwt(authHeader)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Allow-Control-Allow-Methods", "POST")

	var payload BookingReg
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := Validate.Struct(payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	filter := bson.M{"_id": payload.ServiceId}
	ctx, cancle := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancle()
	var service models.Service
	if err := ServiceCollection.FindOne(ctx, filter).Decode(&service); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			http.Error(w, "No service found with given Id", http.StatusBadRequest)
			return
		}
		log.Println("Error finding Service:", err)
		http.Error(w, "An internal server error occurred", http.StatusInternalServerError)
		return
	}
	durationToAdd := time.Duration(service.Duration) * time.Minute
	endTime := payload.StartTime.Add(durationToAdd)

	conflictFilter := bson.M{
		"providerId": service.ProviderId,
		"startTime":  bson.M{"$lt": endTime},
		"endTime":    bson.M{"$gt": payload.StartTime},
	}

	count, err := BookingCollection.CountDocuments(ctx, conflictFilter)
	if err != nil {
		log.Println("Error checking for booking conflicts:", err)
		http.Error(w, "An internal server error occurred", http.StatusInternalServerError)
		return
	}

	if count > 0 {
		http.Error(w, "This time slot is already booked.", http.StatusConflict) // 409 Conflict
		return
	}

	newBooking := models.Booking{
		UserId:       claims.UserId,
		ProviderId:   service.ProviderId,
		ServiceId:    payload.ServiceId,
		StartTime:    payload.StartTime,
		EndTime:      endTime,
		Status:       models.Scheduled, // need to check
		ReminderSent: false,
	}
	_, err = BookingCollection.InsertOne(context.Background(), newBooking)
	if err != nil {
		http.Error(w, "Failed to create booking", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Booking scheduled successfully",
	})
}
