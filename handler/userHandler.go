package handler

import (
	"context"
	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"gopkg.in/rethinkdb/rethinkdb-go.v6"
	"log"
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

	// Calculate total level for all games
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

	// Fetch the game rooms count from RethinkDB
	res, err := rethinkdb.Table("rooms").Run(database.GetRethinkSession())
	if err != nil {
		log.Println("RethinkDB error:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Error fetching game rooms"})
	}
	defer res.Close()

	var rooms []struct {
		GameName string `rethinkdb:"gameName"`
	}

	// Fetch all rooms into the slice
	err = res.All(&rooms)
	if err != nil {
		log.Println("Error fetching rooms:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Error fetching game rooms"})
	}

	// Manually aggregate rooms by gameName
	gameRoomCounts := make(map[string]int)
	for _, room := range rooms {
		gameRoomCounts[room.GameName]++
	}

	// Prepare the user response
	userResponse := fiber.Map{
		"name":         user.Name,
		"avatar":       user.Avatar,
		"games":        user.Games,
		"uniqueId":     user.UniqueID,
		"premium":      user.Premium,
		"premium_ends": user.PremiumEnds,
		"totalLevel":   averageLevel,   // Include total level (average of game levels)
		"roomCounts":   gameRoomCounts, // Include room counts for each game
	}

	// Return the user information and game room counts
	return c.JSON(userResponse)
}
