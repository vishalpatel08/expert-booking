package controller

import (
	auth "BookingPlatfrom/Auth"
	models "BookingPlatfrom/Models"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
)

type RegistrationModel struct {
	FirstName   string `json:"firstName" validate:"required"`
	LastName    string `json:"lastName"`
	PhoneNumber string `json:"phoneNumber"`
	Email       string `json:"email"     validate:"required,email"`
	Password    string `json:"password"  validate:"required,min=6"`
	Role        string `json:"role"      validate:"required,oneof=client provider"`
}

func UserRegistration(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Allow-Control-Allow-Methods", "POST")

	var payload RegistrationModel
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := Validate.Struct(payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	hashedPass, err := bcrypt.GenerateFromPassword([]byte(payload.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal(err)
	}
	user := models.User{
		FirstName:    payload.FirstName,
		LastName:     payload.LastName,
		Email:        payload.Email,
		PhoneNumber:  payload.PhoneNumber,
		PasswordHash: string(hashedPass),
		Role:         models.Role(payload.Role),
		Bookings:     []primitive.ObjectID{},
		CreatedAt:    time.Now().Format("2006-01-02,15:04:05"),
		UpdatedAt:    time.Now().Format("2006-01-02,15:04:05"),
	}

	registered, err := UserCollection.InsertOne(context.Background(), user)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			http.Error(w, "Email already exists", http.StatusConflict)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	newUserId := registered.InsertedID.(primitive.ObjectID)

	tokenString, err := auth.JwtGenerate(newUserId, user.Role)
	if err != nil {
		http.Error(w, "Error in JWT Generation", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "User registered successfully",
		"token":   tokenString,
		"user": map[string]interface{}{
			"_id":         newUserId.Hex(),
			"firstName":   user.FirstName,
			"lastName":    user.LastName,
			"email":       user.Email,
			"phoneNumber": user.PhoneNumber,
			"role":        user.Role,
		},
	})

}
