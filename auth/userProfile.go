package auth

import (
	"context"
	"fmt"
	"math/rand"
	"norex/database"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
)

func UpdateProfile(c *fiber.Ctx) error {
	email := c.Locals("email").(string)
	name := c.FormValue("name")
	gender := c.FormValue("gender")

	if name == "" || gender == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Name and gender are required"})
	}

	collection := database.GetCollection("users")
	update := bson.M{
		"$set": bson.M{
			"name":   name,
			"gender": gender,
			"avatar": generateAvatar(gender),
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
		return fmt.Sprintf("male-%d.jpg", rand.Intn(10)+1)
	}
	return fmt.Sprintf("female-%d.jpg", rand.Intn(10)+1)
}
