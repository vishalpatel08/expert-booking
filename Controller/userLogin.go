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
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
)

type Login struct {
	Email    string `json:"email" validate:"required,email"`
	PassWord string `json:"password" validate:"required"`
}

func UserLogin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var paylod Login
	sendError := func(status int, message string) {
		w.WriteHeader(status)
		json.NewEncoder(w).Encode(map[string]string{"message": message})
	}

	if err := json.NewDecoder(r.Body).Decode(&paylod); err != nil {
		sendError(http.StatusBadRequest, err.Error())
		return
	}

	filter := bson.M{"email": paylod.Email}
	ctx, cancle := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancle()
	var user models.User
	if err := UserCollection.FindOne(ctx, filter).Decode(&user); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			sendError(http.StatusBadRequest, "Invalid Email")
			return
		}
		log.Println("Error finding user:", err)
		http.Error(w, "An internal server error occurred", http.StatusInternalServerError)
		return
	}

	err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(paylod.PassWord))
	if err != nil {
		sendError(http.StatusBadRequest, "Invalid Password")
		return
	}

	tokenString, err := auth.JwtGenerate(user.Id, user.Role)
	if err != nil {
		sendError(http.StatusInternalServerError, "Failed to generate token")
		return
	}
	// Build response containing token and selected user fields (don't return password hash)
	// Include the MongoDB user ID so the frontend can track the current user reliably
	response := map[string]interface{}{
		"message": "Login successful!",
		"token":   tokenString,
		"user": map[string]interface{}{
			"_id":         user.Id.Hex(),
			"firstName":   user.FirstName,
			"lastName":    user.LastName,
			"email":       user.Email,
			"phoneNumber": user.PhoneNumber,
			"role":        user.Role,
		},
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
