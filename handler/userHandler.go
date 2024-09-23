package handler

import (
	"context"
	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"norex/database"
	"norex/models"
)

func GetAuthenticatedUser(c *fiber.Ctx) error {
	// Get the email from the context (set during JWT authentication)
	email := c.Locals("email").(string)

	// Fetch the user details from the database
	var user models.User
	collection := database.GetCollection("users")
	err := collection.FindOne(context.TODO(), bson.M{"email": email}).Decode(&user)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
	}

	totalLevel := 0
	gameCount := len(user.Games) // Assuming user.Games is a map or slice of game stats

	// Sum the levels of all games
	for _, gameStats := range user.Games {
		totalLevel += gameStats.Level // Assuming each game has a 'Level' field in GameStats
	}

	// Calculate the average level
	averageLevel := 0
	if gameCount > 0 {
		averageLevel = totalLevel / gameCount
	}

	// Include the total level (average level) in the userResponse
	userResponse := fiber.Map{
		"name":       user.Name,
		"avatar":     user.Avatar,
		"games":      user.Games,
		"uniqueId":   user.UniqueID,
		"totalLevel": averageLevel, // Include total level (average of game levels)
	}

	// Return the user information
	return c.JSON(userResponse)
}
