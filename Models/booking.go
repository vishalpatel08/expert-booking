package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Status string

const (
	Scheduled Status = "scheduled"
	Completed Status = "completed"
	Cancelled Status = "cancelled"
)

type Booking struct {
	Id           primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty" `
	UserId       primitive.ObjectID `json:"userId,omitempty" bson:"userId,omitempty"`
	ProviderId   primitive.ObjectID `json:"providerId,omitempty" bson:"providerId,omitempty"`
	ServiceId    primitive.ObjectID `json:"serviceId,omitempty" bson:"serviceId,omitempty"`
	StartTime    time.Time          `json:"startTime,omitempty" bson:"startTime,omitempty"`
	EndTime      time.Time          `json:"endTime,omitempty" bson:"endTime,omitempty"`
	Status       Status             `json:"status,omitempty"`
	ReminderSent bool               `json:"reminderSent" bson:"reminderSent"`
}
