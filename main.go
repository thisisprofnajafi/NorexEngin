package main

import (
	"fmt"
	"norex/auth"
	"norex/database"
	"norex/handler"
	"norex/middleware"

	"github.com/gofiber/fiber/v2"
)

func main() {
	// Initialize Fiber app
	app := fiber.New()

	// Connect to MongoDB
	database.Connect()

	// Authentication routes
	app.Post("api/v1/auth/request-code", auth.RequestCode)
	app.Post("api/v1/auth/verify-code", auth.VerifyCode)
	app.Get("/api/v1/auth/validate-token", handler.ValidateToken)

	// Protected routes - Require JWT authentication
	app.Use("/api/v1/protected", auth.JWTProtected())

	// Email verification middleware
	app.Use("/api/v1/protected/profile", middleware.EnsureEmailVerified)

	// Profile update route (name, gender, avatar)
	app.Post("/api/v1/protected/profile", auth.UpdateProfile)

	// Example for admin routes with permission check
	app.Use("/api/v1/admin", middleware.CheckPermissions("manage_users"))
	app.Get("/api/v1/admin/manage", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": "Admin access granted!"})
	})

	// Role CRUD routes with admin protection
	app.Post("/api/v1/protected/roles/", handler.CreateRole, middleware.AdminRequired())
	app.Get("/api/v1/protected/roles/:id", handler.GetRole, middleware.AdminRequired())
	app.Put("/api/v1/protected/roles/:id", handler.UpdateRole, middleware.AdminRequired())
	app.Delete("/api/v1/protected/roles/:id", handler.DeleteRole, middleware.AdminRequired())
	app.Get("/api/v1/protected/roles/", handler.ListRoles, middleware.AdminRequired())

	// Start the server on port 8080 (or 80/443 based on deployment setup)
	err := app.Listen(":8080")
	if err != nil {
		fmt.Printf("Error starting server: %v\n", err)
	}
}
