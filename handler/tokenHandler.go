package handler

import (
	"context"
	"github.com/dgrijalva/jwt-go/v4"
	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"norex/auth"
	"norex/database"
	"norex/models"
)

// ValidateToken is the handler to validate the JWT token and return loggedIn status
// ValidateToken is the handler to validate the JWT token and return loggedIn status
func ValidateToken(c *fiber.Ctx) error {
	// Get the token from the Authorization header
	tokenString := c.Get("Authorization")

	// If no token is provided, return loggedIn as false
	if tokenString == "" {
		return c.JSON(fiber.Map{
			"loggedIn": false,
			"error":    "Missing token",
		})
	}

	// Parse the token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return auth.JWTSecret, nil // Use your jwtSecret from auth package
	})

	// If token is invalid or parsing failed, return loggedIn as false
	if err != nil || !token.Valid {
		return c.JSON(fiber.Map{
			"loggedIn": false,
			"error":    "Invalid or expired token",
		})
	}

	// Retrieve the user's email from the token claims
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return c.JSON(fiber.Map{
			"loggedIn": false,
			"error":    "Invalid token claims",
		})
	}

	userEmail := claims["email"].(string) // Assuming email is part of the token claims

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

	// Check if name or gender is undefined and set require_info accordingly
	requireInfo := user.Name == "" || user.Gender == ""

	// If token is valid, return loggedIn as true and require_info status
	return c.JSON(fiber.Map{
		"loggedIn":     true,
		"require_info": requireInfo,
		"email":        userEmail,
	})
}
