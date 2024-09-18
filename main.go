package main

import (
	"github.com/gofiber/fiber/v2"
	"norex/auth"
	"norex/database"
)

func main() {
	app := fiber.New()

	// MongoDB connection setup
	database.Connect()

	// Routes
	app.Post("api/v1/auth/request-code", auth.RequestCode)
	app.Post("api/v1/auth/verify-code", auth.VerifyCode)

	// Protected API routes
	app.Use("/api/v1/protected", auth.JWTProtected())
	//app.Get("/api/v1/protected/start-game", someHandler) // Protected route

	app.Listen(":3000")
}
