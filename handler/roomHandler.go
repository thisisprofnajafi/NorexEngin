package handler

import (
	"context"
	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	rethink "gopkg.in/rethinkdb/rethinkdb-go.v6"
	"math/rand"
	"norex/database" // Adjust the import path according to your project structure
	"norex/models"
)

func generateRoomID() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 9)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

func CreateRoom(c *fiber.Ctx) error {
	var room struct {
		RoomID       string `json:"roomId"`
		GameName     string `json:"gameName"`
		IsLocked     bool   `json:"isLocked"`
		RoomPassword string `json:"roomPassword"`
		VoiceChatOn  bool   `json:"voiceChatOn"`
		TextChatOn   bool   `json:"textChatOn"`
		MinLevel     int    `json:"minLevel"`
		UserID       string `json:"userId"`
		Avatar       string `json:"avatar"`
		Name         string `json:"name"`
	}

	// Generate a unique RoomID
	room.RoomID = generateRoomID()

	// Get the user's email from c.Locals
	email := c.Locals("email").(string)

	// Fetch the user ID from MongoDB using the email
	var user models.User
	if err := database.GetCollection("users").FindOne(context.TODO(), bson.M{"email": email}).Decode(&user); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to find user"})
	}
	room.UserID = user.ID.Hex()
	room.Avatar = user.Avatar
	room.Name = user.Name

	// Parse the JSON body
	if err := c.BodyParser(&room); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	// Get the RethinkDB session
	session := database.GetRethinkSession()

	// Insert the room into RethinkDB
	_, err := rethink.Table("rooms").Insert(room).Run(session)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create room " + err.Error()})
	}

	return c.JSON(fiber.Map{"message": "Room created successfully", "roomId": room.RoomID, "gameName": room.GameName})
}
