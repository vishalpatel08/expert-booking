package controller

import (
	models "BookingPlatfrom/Models"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

type SendReminderRequest struct {
	BookingID string `json:"bookingId"`
	UserID    string `json:"userId"`
	UserEmail string `json:"userEmail"`
	Message   string `json:"message"`
}

func SendReminders() {
	log.Println("Running reminder job...")

	now := time.Now().UTC()
	fmt.Println(now)
	startTimeWindow := now.Add(30 * time.Minute)
	endTimeWindow := now.Add(31 * time.Minute)

	filter := bson.M{
		"startTime":    bson.M{"$gte": startTimeWindow, "$lt": endTimeWindow},
		"status":       models.Scheduled,
		"reminderSent": false,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cursor, err := BookingCollection.Find(ctx, filter)
	if err != nil {
		log.Printf("Error finding bookings for reminders: %v\n", err)
		return
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var booking models.Booking
		if err := cursor.Decode(&booking); err != nil {
			log.Printf("Error decoding booking: %v\n", err)
			continue
		}

		var user models.User
		if err := UserCollection.FindOne(ctx, bson.M{"_id": booking.UserId}).Decode(&user); err != nil {
			log.Printf("Could not find user %s for reminder", booking.UserId.Hex())
			continue
		}

		reqPayload := SendReminderRequest{
			BookingID: booking.Id.Hex(),
			UserID:    booking.UserId.Hex(),
			UserEmail: user.Email,
			Message:   "Your appointment is in 30 minutes.",
		}

		payloadBytes, err := json.Marshal(reqPayload)
		if err != nil {
			log.Printf("Error marshaling request payload: %v", err)
			continue
		}

		resp, err := http.Post("http://localhost:50052/send-reminder", "application/json", bytes.NewBuffer(payloadBytes))
		if err != nil {
			log.Printf("Failed to call notification service: %v", err)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			log.Printf("Successfully dispatched reminder for booking %s", booking.Id.Hex())
			updateFilter := bson.M{"_id": booking.Id}
			update := bson.M{"$set": bson.M{"reminderSent": true}}
			_, err := BookingCollection.UpdateOne(ctx, updateFilter, update)
			if err != nil {
				log.Printf("Failed to update reminderSent status for booking ID %s: %v\n", booking.Id.Hex(), err)
			}
		} else {
			log.Printf("Notification service returned an error: %s", resp.Status)
		}
	}
}
