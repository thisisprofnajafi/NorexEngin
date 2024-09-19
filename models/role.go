package models

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"norex/database"
	"time"
)

type Role struct {
	ID            primitive.ObjectID `bson:"_id,omitempty"`
	Name          string             `bson:"name"`
	Permissions   []string           `bson:"permissions"`
	SessionExpiry time.Duration      `bson:"session_expiry"`
}

func CreateRoles() {
	collection := database.GetCollection("roles")

	roles := []Role{
		{
			Name:          "user",
			Permissions:   []string{"read", "play_game"},
			SessionExpiry: 365 * 24 * time.Hour, // 1 year
		},
		{
			Name:          "premium_user",
			Permissions:   []string{"read", "play_game", "access_premium"},
			SessionExpiry: 365 * 24 * time.Hour, // 1 year
		},
		{
			Name:          "admin",
			Permissions:   []string{"read", "play_game", "access_premium", "manage_users", "delete_content"},
			SessionExpiry: 24 * time.Hour, // 1 day
		},
	}

	for _, role := range roles {
		_, err := collection.InsertOne(context.TODO(), role)
		if err != nil {
			fmt.Println("Error inserting roles:", err)
		}
	}
}
