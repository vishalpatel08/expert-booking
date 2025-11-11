package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ProviderProfile struct {
	Id             primitive.ObjectID   `json:"_id,omitempty" bson:"_id,omitempty"`
	UserId         primitive.ObjectID   `json:"userId" bson:"userId"`
	Title          string               `json:"title"`
	Domain         string               `json:"domain"`
	Qualifications string               `json:"qualifications"`
	ServiceIds     []primitive.ObjectID `json:"serviceIds,omitempty" bson:"serviceIds,omitempty"`
}

type Service struct {
	Id         primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty"`
	ProviderId primitive.ObjectID `json:"providerId" bson:"providerId"`
	Title      string             `json:"title"`
	Duration   int64              `json:"duration"`
	Price      float64            `json:"price"`
}
