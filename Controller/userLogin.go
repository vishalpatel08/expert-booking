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
	w.Header().Set("Allow-Control-Allow-Methods", "POST")
	var paylod Login
	if err := json.NewDecoder(r.Body).Decode(&paylod); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	filter := bson.M{"email": paylod.Email}
	ctx, cancle := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancle()
	var user models.User
	if err := UserCollection.FindOne(ctx, filter).Decode(&user); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			http.Error(w, "Invalid Email", http.StatusBadRequest)
			return
		}
		log.Println("Error finding user:", err)
		http.Error(w, "An internal server error occurred", http.StatusInternalServerError)
		return
	}

	err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(paylod.PassWord))
	if err != nil {
		http.Error(w, "Invalid Password", http.StatusBadRequest)
		return
	}

	tokenString, err := auth.JwtGenerate(user.Id, user.Role)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{
		"message": "Login successful!",
		"token":   tokenString,
	})
}
