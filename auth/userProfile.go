package auth

import (
	"context"
	"fmt"
	"math/rand"
	"norex/database"
	"norex/models"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
)

func UpdateProfile(c *fiber.Ctx) error {

	email := c.Locals("email").(string)

	var body struct {
		Name   string `json:"name"`
		Gender string `json:"gender"`
	}

	// Parse the JSON body
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	name := body.Name
	gender := body.Gender

	if name == "" || gender == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Name and gender are required"})
	}

	defaultGames := map[string]models.GameStats{
		"uno":         {Wins: 0, Level: 1},
		"spades":      {Wins: 0, Level: 1},
		"go_fish":     {Wins: 0, Level: 1},
		"euchre":      {Wins: 0, Level: 1},
		"hearts":      {Wins: 0, Level: 1},
		"crazy_eight": {Wins: 0, Level: 1},
		"chess":       {Wins: 0, Level: 1},
		"othello":     {Wins: 0, Level: 1},
		"go":          {Wins: 0, Level: 1},
		"checkers":    {Wins: 0, Level: 1},
		"battleship":  {Wins: 0, Level: 1},
		"image_match": {Wins: 0, Level: 1},
	}

	collection := database.GetCollection("users")
	update := bson.M{
		"$set": bson.M{
			"name":   name,
			"gender": gender,
			"avatar": generateAvatar(gender),
			"games":  defaultGames,
		},
	}

	_, err := collection.UpdateOne(context.TODO(), bson.M{"email": email}, update)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update profile"})
	}

	return c.JSON(fiber.Map{"message": "Profile updated successfully"})
}

func generateAvatar(gender string) string {
	if gender == "Male" {
		return fmt.Sprintf("man-%d.jpg", rand.Intn(10)+1)
	}
	return fmt.Sprintf("lady-%d.jpg", rand.Intn(10)+1)
}
