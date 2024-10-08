package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

type GameStats struct {
	Wins  int `bson:"wins" json:"wins"`
	Level int `bson:"level" json:"level"`
}

type User struct {
	ID                primitive.ObjectID   `bson:"_id,omitempty"`
	Email             string               `bson:"email"`
	VerificationCode  string               `bson:"verification_code"`
	CodeExpiryTime    time.Time            `bson:"code_expiry_time"`
	AttemptCount      int                  `bson:"attempt_count"`
	BanUntil          time.Time            `bson:"ban_until"`
	Name              string               `bson:"name,omitempty"`
	Gender            string               `bson:"gender,omitempty"`
	UniqueID          string               `bson:"unique_id,omitempty"`
	Avatar            string               `bson:"avatar,omitempty"`
	VerifiedEmailDate time.Time            `bson:"verified_email_date,omitempty"`
	Premium           bool                 `bson:"premium,omitempty"`
	PremiumEnds       time.Time            `bson:"premium_ends,omitempty"`
	Role              string               `bson:"role"` // Add this field
	Games             map[string]GameStats `bson:"games" json:"games"`
}
