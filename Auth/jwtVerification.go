package auth

import (
	"errors"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

func VerifyJwt(authHeader string) (*Claims, error) {
	if authHeader == "" {
		return nil, errors.New("missing authorization header")
	}
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(os.Getenv("JWT_SECRET_KEY")), nil
	})
	if err != nil || !token.Valid {
		return nil, err
	}
	return claims, nil
}
