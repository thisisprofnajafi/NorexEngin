package middleware

import (
	"context"
	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"norex/database"
	"norex/models"
	"time"
)

func EnsureEmailVerified(c *fiber.Ctx) error {
	// Get the token from the Authorization header
	tokenString := c.Get("Authorization")
	if tokenString == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Missing token"})
	}

	// Fetch the session using the token from the database
	var session models.Session
	collection := database.GetCollection("sessions")
	err := collection.FindOne(context.TODO(), bson.M{"token": tokenString}).Decode(&session)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid or expired session token"})
	}

	// Check if the session is expired
	if session.ExpiresAt.Before(time.Now()) {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Session has expired"})
	}

	// Fetch the user associated with the session
	var user models.User
	userCollection := database.GetCollection("users")
	err = userCollection.FindOne(context.TODO(), bson.M{"_id": session.UserID}).Decode(&user)
	if err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "User not found"})
	}

	// Check if the user's email is verified
	if user.VerifiedEmailDate.IsZero() {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Email not verified"})
	}

	// If email is verified, allow the request to proceed
	return c.Next()
}

func NameGenderCheck(c *fiber.Ctx) error {
	email := c.Locals("email").(string)
	var user models.User
	collection := database.GetCollection("users")
	err := collection.FindOne(context.TODO(), bson.M{"email": email}).Decode(&user)
	if err != nil || user.Name == "" || user.Gender == "" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Name and gender required"})
	}
	return c.Next()
}

func CheckPermissions(requiredPermission string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userRole := c.Locals("role").(string)

		// Fetch role and permissions from the database
		var role models.Role
		err := database.GetCollection("roles").FindOne(context.TODO(), bson.M{"name": userRole}).Decode(&role)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to retrieve role"})
		}

		// Check if the role has the required permission
		for _, perm := range role.Permissions {
			if perm == requiredPermission {
				return c.Next() // Permission granted
			}
		}

		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Insufficient permissions"})
	}
}
