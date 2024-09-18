package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

type User struct {
	ID               primitive.ObjectID `bson:"_id,omitempty"`
	Email            string             `bson:"email"`
	VerificationCode string             `bson:"verification_code"`
	CodeExpiryTime   time.Time          `bson:"code_expiry_time"`
	AttemptCount     int                `bson:"attempt_count"`
	BanUntil         time.Time          `bson:"ban_until"`
}
