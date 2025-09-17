package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type TimeSlot struct {
	IsAvailable bool   `json:"isAvailable" bson:"isAvailable"`
	StartTime   string `json:"startTime" bson:"startTime"`
	EndTime     string `json:"endTime" bson:"endTime"`
}

type ProviderSchedule struct {
	ID         primitive.ObjectID  `json:"_id,omitempty" bson:"_id,omitempty"`
	ProviderID primitive.ObjectID  `json:"providerId" bson:"providerId"`
	Week       map[string]TimeSlot `json:"week" bson:"week"`
}
