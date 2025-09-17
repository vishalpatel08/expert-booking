package auth

import (
	models "BookingPlatfrom/Models"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Claims struct {
	UserId primitive.ObjectID `json:"userId"`
	Role   models.Role        `json:"role"`
	jwt.RegisteredClaims
}

func JwtGenerate(userId primitive.ObjectID, role models.Role) (string, error) {
	var jwtKey = []byte(os.Getenv("JWT_SECRET_KEY"))
	expirationTime := time.Now().Add(24 * time.Hour)

	claims := &Claims{
		UserId: userId,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		return "", err
	}
	return tokenString, nil
}
