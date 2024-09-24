package main

import (
	"fmt"
	"github.com/goccy/go-json"
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"log"
	"norex/auth"
	"norex/database"
	"norex/handler"
	"norex/middleware"
)

func main() {
	// Initialize Fiber app
	app := fiber.New(fiber.Config{
		JSONEncoder: json.Marshal,
		JSONDecoder: json.Unmarshal,
	})

	// Connect to MongoDB
	database.Connect()
	database.ConnectRethinkDB()

	//delete all the rows
	//rethink.Table("rooms").Delete().RunWrite(database.GetRethinkSession())

	api := app.Group("/api/v1")

	// Authentication routes
	api.Post("/auth/request-code", auth.RequestCode)
	api.Post("/auth/verify-code", auth.VerifyCode)
	api.Get("/auth/validate-token", handler.ValidateToken)

	// Protected routes - Require JWT authentication
	protected := api.Group("/protected", auth.JWTProtected())

	// Email verification middleware
	protected.Use(middleware.EnsureEmailVerified)

	// Profile routes
	protected.Post("/profile", auth.UpdateProfile)
	protected.Get("/user/profile", handler.GetAuthenticatedUser)

	// Admin routes
	admin := api.Group("/admin", middleware.CheckPermissions("manage_users"))
	admin.Get("/manage", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": "Admin access granted!"})
	})

	// Role CRUD routes with admin protection
	role := protected.Group("/roles", middleware.AdminRequired())
	role.Post("/", handler.CreateRole)
	role.Get("/:id", handler.GetRole)
	role.Put("/:id", handler.UpdateRole)
	role.Delete("/:id", handler.DeleteRole)
	role.Get("/", handler.ListRoles)

	// Room creation route
	protected.Post("/new/room", handler.CreateRoom)

	webSocket := api.Use("/ws", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			c.Locals("allowed", true)
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})

	webSocket.Get("/all-games", websocket.New(func(c *websocket.Conn) {
		// c.Locals is added to the *websocket.Conn
		log.Println(c.Locals("allowed"))  // true
		log.Println(c.Params("id"))       // 123
		log.Println(c.Query("v"))         // 1.0
		log.Println(c.Cookies("session")) // ""

		// websocket.Conn bindings https://pkg.go.dev/github.com/fasthttp/websocket?tab=doc#pkg-index
		var (
			mt  int
			msg []byte
			err error
		)
		for {
			if mt, msg, err = c.ReadMessage(); err != nil {
				log.Println("read:", err)
				break
			}
			log.Printf("recv: %s", msg)

			if err = c.WriteMessage(mt, msg); err != nil {
				log.Println("write:", err)
				break
			}
		}

	}))

	// Start the server on port 8080 (or 80/443 based on deployment setup)
	err := app.Listen(":9990")
	if err != nil {
		fmt.Printf("Error starting server: %v\n", err)
	}
}

// WebSocket handler function
func AllGamesSocket(c *websocket.Conn) {
	var (
		mt  int
		msg []byte
		err error
	)
	for {
		// Read the WebSocket message
		if mt, msg, err = c.ReadMessage(); err != nil {
			log.Println("read error:", err)
			break
		}

		// Echo the message back to the client
		log.Printf("recv: %s", msg)
		if err = c.WriteMessage(mt, msg); err != nil {
			log.Println("write error:", err)
			break
		}
	}
}
