package handler

import (
	"context"
	"github.com/dgrijalva/jwt-go/v4"
	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"norex/auth"
	"norex/database"
	"norex/models"
	"strings"
)

// ValidateToken is the handler to validate the JWT token and return loggedIn status
func ValidateToken(c *fiber.Ctx) error {
	// Get the token from the Authorization header
	tokenString := c.Get("Authorization")
	if tokenString == "" {
		return c.JSON(fiber.Map{
			"loggedIn": false,
			"error":    "Missing token",
		})
	}

	// Remove "Bearer " prefix if present
	if strings.HasPrefix(tokenString, "Bearer ") {
		tokenString = strings.TrimPrefix(tokenString, "Bearer ")
	}

	// Parse the token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return auth.JWTSecret, nil // Use your jwtSecret from the auth package
	})

	// If the token is invalid or expired, return loggedIn as false
	if err != nil || !token.Valid {
		return c.JSON(fiber.Map{
			"loggedIn": false,
			"error":    "Invalid or expired token",
		})
	}

	// Retrieve claims from the token
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return c.JSON(fiber.Map{
			"loggedIn": false,
			"error":    "Invalid token claims",
		})
	}

	// Extract email from claims
	userEmail, ok := claims["email"].(string)
	if !ok || userEmail == "" {
		return c.JSON(fiber.Map{
			"loggedIn": false,
			"error":    "Email not found in token",
		})
	}

	// Fetch the user from the database using the email
	collection := database.GetCollection("users")
	var user models.User
	err = collection.FindOne(context.TODO(), bson.M{"email": userEmail}).Decode(&user)
	if err != nil {
		return c.JSON(fiber.Map{
			"loggedIn": false,
			"error":    "User not found",
		})
	}

	// Check if the user's name or gender is missing, requiring additional information
	requireInfo := user.Name == "" || user.Gender == ""

	// If token is valid, return loggedIn as true and include require_info status
	return c.JSON(fiber.Map{
		"loggedIn":     true,
		"require_info": requireInfo,
		"email":        userEmail,
	})
}
