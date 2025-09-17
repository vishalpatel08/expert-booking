package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type Role string

const (
	RoleClient   Role = "client"
	RoleProvider Role = "provider"
)

type User struct {
	Id           primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty"`
	FirstName    string             `json:"firstName,omitempty"`
	LastName     string             `json:"lastName"`
	Email        string             `json:"email,omitempty"`
	PhoneNumber  string             `json:"phoneNumber"`
	PasswordHash string             `json:"passwordHash,omitempty"`
	Role         Role               `json:"role" bson:"role"`
	CreatedAt    string             `json:"createdAt"`
	UpdatedAt    string             `json:"updatedAt"`
}
