package handler

import (
	"github.com/dgrijalva/jwt-go/v4"
	"github.com/gofiber/fiber/v2"
	"norex/auth"
)

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

	// If token is valid, return loggedIn as true
	return c.JSON(fiber.Map{
		"loggedIn": true,
	})
}
