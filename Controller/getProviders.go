package controller

import (
	models "BookingPlatfrom/Models"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ProviderResponse struct {
	ID             primitive.ObjectID   `json:"id" bson:"_id,omitempty"`
	UserId         primitive.ObjectID   `json:"userId" bson:"userId"`
	Title          string               `json:"title"`
	Domain         string               `json:"domain"`
	Qualifications string               `json:"qualifications"`
	FirstName      string               `json:"firstName,omitempty"`
	LastName       string               `json:"lastName,omitempty"`
	Email          string               `json:"email,omitempty"`
	PhoneNumber    string               `json:"phoneNumber,omitempty"`
	ServiceIds     []primitive.ObjectID `json:"serviceIds,omitempty"`
	Services       []models.Service     `json:"services,omitempty"`
}

func GetProviders(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, err := ProviderCollection.Find(ctx, bson.M{})
	if err != nil {
		log.Println("Error fetching providers:", err)
		http.Error(w, "Failed to fetch providers", http.StatusInternalServerError)
		return
	}
	defer cursor.Close(ctx)

	var providers []models.ProviderProfile
	if err := cursor.All(ctx, &providers); err != nil {
		log.Println("Error decoding providers:", err)
		http.Error(w, "Failed to decode providers", http.StatusInternalServerError)
		return
	}

	resp := make([]ProviderResponse, 0, len(providers))

	for _, p := range providers {
		pr := ProviderResponse{
			ID:             p.Id,
			UserId:         p.UserId,
			Title:          p.Title,
			Domain:         p.Domain,
			Qualifications: p.Qualifications,
			ServiceIds:     p.ServiceIds,
		}

		var user models.User
		if err := UserCollection.FindOne(ctx, bson.M{"_id": p.UserId}).Decode(&user); err == nil {
			pr.FirstName = user.FirstName
			pr.LastName = user.LastName
			pr.Email = user.Email
			pr.PhoneNumber = user.PhoneNumber
		}

		svcCursor, err := ServiceCollection.Find(ctx, bson.M{"providerId": p.UserId})
		if err == nil {
			var services []models.Service
			if err := svcCursor.All(ctx, &services); err == nil {
				pr.Services = services
			}
			svcCursor.Close(ctx)
		}

		resp = append(resp, pr)
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"providers": resp,
	})
}
