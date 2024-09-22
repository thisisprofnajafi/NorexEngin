package middleware

import (
	"context"
	"github.com/dgrijalva/jwt-go/v4"
	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"norex/auth"
	"norex/database"
	"norex/models"
)

func EnsureEmailVerified(c *fiber.Ctx) error {
	// Get the token from the Authorization header
	tokenString := c.Get("Authorization")
	if tokenString == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Missing token"})
	}

	// Parse the token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Provide the secret key for validation (use your jwtSecret from the auth package)
		return auth.JWTSecret, nil
	})

	// Check if the token is valid
	if err != nil || !token.Valid {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid or expired token"})
	}

	// Extract claims (assuming the token contains user information like email or user ID)
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Failed to parse token claims"})
	}

	// Get user email or user ID from the claims (adjust based on your token structure)
	email, ok := claims["email"].(string) // Or use "user_id" if you store user ID
	if !ok || email == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid token payload"})
	}

	// Fetch the user from the database
	var user models.User
	collection := database.GetCollection("users")
	err = collection.FindOne(context.TODO(), bson.M{"email": email}).Decode(&user)
	if err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "User not found"})
	}

	// Check if the user has a valid VerifiedEmailDate
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
