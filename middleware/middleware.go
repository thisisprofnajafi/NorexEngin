package middleware

import (
	"context"
	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"norex/database"
	"norex/models"
)

func EnsureEmailVerified(c *fiber.Ctx) error {
	// Get the email from the request context
	email, ok := c.Locals("email").(string)
	if !ok || email == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Email not found in request"})
	}

	// Fetch the user from the database
	var user models.User
	collection := database.GetCollection("users")
	err := collection.FindOne(context.TODO(), bson.M{"email": email}).Decode(&user)
	if err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "User not found"})
	}

	// Check if the user has a valid VerifiedEmailDate
	if user.VerifiedEmailDate.IsZero() {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Email not verified"})
	}

	// If email is verified, allow request to proceed
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
