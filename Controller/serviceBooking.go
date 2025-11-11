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

type BookingReg struct {
	ServiceId string    `json:"serviceId,omitempty" validate:"required"`
	StartTime time.Time `json:"startTime,omitempty" validate:"required"`
}

func ServiceBooking(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	claims, err := auth.VerifyJwt(authHeader)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Methods", "POST")

	var payload BookingReg
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	fmt.Println(" Data : ", payload)
	if err := Validate.Struct(payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	sid, err := primitive.ObjectIDFromHex(payload.ServiceId)
	if err != nil {
		http.Error(w, "invalid serviceId", http.StatusBadRequest)
		return
	}
	filter := bson.M{"_id": sid}
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
		ServiceId:    sid,
		StartTime:    payload.StartTime,
		EndTime:      endTime,
		Status:       models.Scheduled, // need to check
		ReminderSent: false,
	}
	res, err := BookingCollection.InsertOne(context.Background(), newBooking)
	if err != nil {
		http.Error(w, "Failed to create booking", http.StatusInternalServerError)
		return
	}
	if oid, ok := res.InsertedID.(primitive.ObjectID); ok {
		_, _ = UserCollection.UpdateOne(
			context.Background(),
			bson.M{"_id": claims.UserId},
			bson.M{"$push": bson.M{"bookingIds": oid}},
		)
	}
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Booking scheduled successfully",
	})
}

// GetMyBookings returns bookings for the authenticated user
func GetMyBookings(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	claims, err := auth.VerifyJwt(authHeader)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	w.Header().Set("Content-Type", "application/json")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, err := BookingCollection.Find(ctx, bson.M{"userId": claims.UserId})
	if err != nil {
		log.Println("Error fetching bookings:", err)
		http.Error(w, "Failed to fetch bookings", http.StatusInternalServerError)
		return
	}
	defer cursor.Close(ctx)

	var bookings []models.Booking
	if err := cursor.All(ctx, &bookings); err != nil {
		log.Println("Error decoding bookings:", err)
		http.Error(w, "Failed to decode bookings", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"bookings": bookings,
	})
}
