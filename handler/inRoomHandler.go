package handler

import (
	"context"
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"gopkg.in/rethinkdb/rethinkdb-go.v6"
	"log"
	"norex/database"
	"norex/models"
)

var roomClients = make(map[string]map[*websocket.Conn]bool) // Map game_id -> clients

func HandleGameRoom(c *websocket.Conn) {
	gameID := c.Params("game_id")
	userEmail := c.Locals("email").(string)

	// Fetch user info from MongoDB based on their email
	var user struct {
		Name   string `bson:"name"`
		Avatar string `bson:"avatar"`
	}
	collection := database.GetCollection("users")
	err := collection.FindOne(context.Background(), bson.M{"email": userEmail}).Decode(&user)
	if err != nil {
		log.Printf("Failed to fetch user data for email %s: %v", userEmail, err)
		return
	}

	// Add the current connection to the room clients
	if roomClients[gameID] == nil {
		roomClients[gameID] = make(map[*websocket.Conn]bool)
	}
	roomClients[gameID][c] = true

	// Broadcast that a new user has joined the room
	broadcastToRoom(gameID, "new_user", fiber.Map{
		"userName": user.Name,
		"avatar":   user.Avatar,
		"email":    userEmail,
	})

	defer func() {
		// Check if the user is still connected before proceeding
		if _, ok := roomClients[gameID][c]; ok {
			// Remove the user from the room clients on disconnect
			delete(roomClients[gameID], c)
			c.Close()

			// Broadcast that the user has left the room
			broadcastToRoom(gameID, "user_left", fiber.Map{
				"userName": user.Name,
				"avatar":   user.Avatar,
				"email":    userEmail,
			})

			// Check if the current user is the room owner by querying RethinkDB
			isOwner, err := checkIfUserIsOwner(gameID, userEmail)
			if err != nil {
				log.Println("Error checking owner in RethinkDB:", err)
				return
			}

			// If the owner is leaving, delete the room
			if isOwner {
				// Broadcast that the room is being deleted
				broadcastToRoom(gameID, "room_deleted", fiber.Map{
					"gameID": gameID,
					"owner":  userEmail,
				})

				// Delete the room data from RethinkDB
				deleteRoomFromDatabase(gameID)

				// Clean up the room clients data
				delete(roomClients, gameID)
			}
		}
	}()

	for {
		_, msg, err := c.ReadMessage()
		if err != nil {
			// Only log errors that are not normal closure
			if websocket.IsUnexpectedCloseError(err, websocket.CloseNormalClosure) {
				log.Println("WebSocket error:", err)
			}
			delete(roomClients[gameID], c)
			c.Close()
			break
		}
		// Handle incoming messages or other events
		log.Println("Message received:", string(msg))
	}
}

func WatchRoomDelete() {
	cursor, err := rethinkdb.Table("rooms").Changes().Run(database.GetRethinkSession())
	if err != nil {
		log.Println("RethinkDB error:", err)
		return
	}
	defer cursor.Close()

	var change map[string]interface{}
	for cursor.Next(&change) {
		// Check if a room was deleted
		if change["old_val"] != nil && change["new_val"] == nil {
			// Room deleted event
			roomID := change["old_val"].(map[string]interface{})["id"].(string)
			ownerEmail := change["old_val"].(map[string]interface{})["userEmail"].(string) // Get the owner email
			broadcastToRoom(roomID, "room_deleted", fiber.Map{
				"roomID": roomID,
				"owner":  ownerEmail, // Broadcast the owner's email or name if needed
			})
		}
	}

	if err := cursor.Err(); err != nil {
		log.Println("Cursor error:", err)
	}
}

// Helper function to check if the user is the room owner
func checkIfUserIsOwner(gameID, userEmail string) (bool, error) {
	var room struct {
		UserEmail string `rethinkdb:"userEmail"`
	}

	// Get the room document from the rooms table using its ID
	res, err := rethinkdb.Table("rooms").Get(gameID).Run(database.GetRethinkSession())
	if err != nil {
		return false, err
	}
	defer res.Close()

	// Decode the result into the room struct
	if err := res.One(&room); err != nil {
		if err == rethinkdb.ErrEmptyResult {
			return false, nil // Room not found
		}
		return false, err
	}

	// Return true if the user's email matches the room owner's email
	return room.UserEmail == userEmail, nil
}

// Helper function to delete the room from RethinkDB
func deleteRoomFromDatabase(gameID string) {
	_, err := rethinkdb.Table("rooms").Get(gameID).Delete().RunWrite(database.GetRethinkSession())
	if err != nil {
		log.Println("Error deleting room from RethinkDB:", err)
	} else {
		log.Println("Room deleted:", gameID)
	}
}

// Helper function to broadcast events
func broadcastToRoom(gameID string, event string, message fiber.Map) {
	for conn := range roomClients[gameID] {
		if err := conn.WriteJSON(fiber.Map{
			"type": event,
			"data": message,
		}); err != nil {
			log.Println("Error sending message to room:", err)
			conn.Close()
			delete(roomClients[gameID], conn) // Clean up disconnected client
		}
	}
}

// Broadcast when a user subscribes/unsubscribes
func broadcastUserEvent(gameID string, eventName string, user fiber.Map) {
	broadcastToRoom(gameID, eventName, user)
}

// ===================== api

func ParticipateInGame(c *fiber.Ctx) error {
	gameID := c.Params("game_id")
	// Logic to add user to the game
	// Broadcast event: "someone participated"
	broadcastUserEvent(gameID, "user_participated", fiber.Map{
		"userName": "someUser",
		"avatar":   "someAvatar",
		"level":    1, // Get user level from the database
	})
	return c.JSON(fiber.Map{"status": "user participated"})
}

func CancelParticipation(c *fiber.Ctx) error {
	gameID := c.Params("game_id")
	// Logic to remove user from game (but not owner)
	// Broadcast event: "someone canceled participation"
	broadcastUserEvent(gameID, "user_canceled", fiber.Map{
		"userName": "someUser",
		"avatar":   "someAvatar",
		"level":    1, // Get user level from the database
	})
	return c.JSON(fiber.Map{"status": "participation canceled"})
}

func SendMessage(c *fiber.Ctx) error {
	// Get the game ID from URL params
	gameID := c.Params("game_id")

	// Get the user's email from the request context
	userEmail := c.Locals("email").(string)

	// Fetch user info from MongoDB based on their email
	var user struct {
		Name   string `bson:"name"`
		Avatar string `bson:"avatar"`
	}

	// Replace "users" with your MongoDB collection name
	collection := database.GetCollection("users") // Assuming this is how you access MongoDB collection
	err := collection.FindOne(c.Context(), bson.M{"email": userEmail}).Decode(&user)
	if err != nil {
		// Handle the error if the user is not found or another error occurs
		log.Printf("Failed to fetch user data for email %s: %v", userEmail, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "User not found"})
	}

	// Broadcast the message to the room with user's info
	broadcastToRoom(gameID, "new_message", fiber.Map{
		"userName": user.Name,
		"avatar":   user.Avatar,
		"message":  c.FormValue("message"),
	})

	// Return a success response
	return c.JSON(fiber.Map{"status": "message sent"})
}

func StartGame(c *fiber.Ctx) error {
	gameID := c.Params("game_id")
	userEmail := c.Locals("email").(string) // Assuming the user's email is set in the context during authentication

	// Check if the user is the owner of the room
	isOwner, err := checkIfUserIsOwner(gameID, userEmail)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Internal server error"})
	}

	if !isOwner {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Only the room owner can start the game"})
	}

	// Fetch the game name from the rooms table
	var room struct {
		GameName string `rethinkdb:"gameName"`
	}

	cursor, err := rethinkdb.Table("rooms").Get(gameID).Field("gameName").Run(database.GetRethinkSession())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch room information"})
	}

	// Fetch the game name
	if err = cursor.One(&room.GameName); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to read room information"})
	}

	// Create a new entry in the games table
	gameEntry := map[string]interface{}{
		"roomId":       gameID,
		"gameName":     room.GameName, // Use the game name retrieved from the rooms table
		"status":       "started",
		"ownerId":      userEmail,           // Store the owner's email as the ownerId
		"winnerId":     nil,                 // Placeholder for the winner; can be updated later
		"participates": []string{userEmail}, // Start with the owner as a participant
	}

	// Insert the new game entry into the games table
	_, err = rethinkdb.Table("games").Insert(gameEntry).RunWrite(database.GetRethinkSession())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to start the game"})
	}

	// Broadcast event: "game started"
	broadcastToRoom(gameID, "game_started", fiber.Map{
		"owner": userEmail, // or fetch the owner's name if necessary
	})

	return c.JSON(fiber.Map{"status": "game started"})
}

func GetRoomInformation(c *fiber.Ctx) error {
	gameID := c.Params("game_id")

	// Define a struct to hold the room information
	var room struct {
		UserEmail string `rethinkdb:"userEmail"`
		GameID    string `rethinkdb:"gameID"`   // Assuming the room table in RethinkDB has gameID field
		Settings  string `rethinkdb:"settings"` // Assuming you have a field for game settings
	}

	// Fetch the room from the database
	res, err := rethinkdb.Table("rooms").Get(gameID).Run(database.GetRethinkSession())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not retrieve room"})
	}
	defer res.Close()

	// Decode the result into the room struct
	if err := res.One(&room); err != nil {
		if err == rethinkdb.ErrEmptyResult {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Room not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not decode room information"})
	}

	// Now we need to get the owner's details from MongoDB
	var owner models.User // Assuming you have a User model to hold user information
	collection := database.GetCollection("users")

	// Fetch the owner's information from the users collection using the email
	err = collection.FindOne(context.TODO(), bson.M{"email": room.UserEmail}).Decode(&owner)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not retrieve owner information"})
	}

	// Fetch the game ID from the room and then get the level for that specific game
	gameLevel := owner.Games[room.GameID].Level

	// Prepare the room information response
	roomInfo := fiber.Map{
		"ownerName": owner.Name,
		"level":     gameLevel,     // Owner's level for the specific game
		"avatar":    owner.Avatar,  // Owner's avatar
		"settings":  room.Settings, // Room settings from RethinkDB
	}

	return c.JSON(roomInfo)
}

func WatchRoomUserParticipation() {
	cursor, err := rethinkdb.Table("participated").Changes().Run(database.GetRethinkSession())
	if err != nil {
		log.Println("RethinkDB error:", err)
		return
	}
	defer cursor.Close()

	var change map[string]interface{}
	for cursor.Next(&change) {
		if change["new_val"] != nil {
			// User participated in a room
			userID := change["new_val"].(map[string]interface{})["userID"].(string)
			roomID := change["new_val"].(map[string]interface{})["roomID"].(string)
			userName := change["new_val"].(map[string]interface{})["userName"].(string)
			userAvatar := change["new_val"].(map[string]interface{})["userAvatar"].(string)
			userLevel := change["new_val"].(map[string]interface{})["userLevel"].(int) // Assuming level is an int

			// Broadcast participation details to the room
			broadcastToRoom(roomID, "user_participated", fiber.Map{
				"userID":     userID,
				"userName":   userName,
				"userAvatar": userAvatar,
				"userLevel":  userLevel,
			})
		}

		if change["old_val"] != nil {
			// User canceled participation
			userID := change["old_val"].(map[string]interface{})["userID"].(string)
			roomID := change["old_val"].(map[string]interface{})["roomID"].(string)

			// Broadcast cancellation details to the room
			broadcastToRoom(roomID, "user_canceled_participation", fiber.Map{
				"userID": userID,
			})
		}
	}

	if err := cursor.Err(); err != nil {
		log.Println("Cursor error:", err)
	}
}

func WatchRoomMessages() {
	cursor, err := rethinkdb.Table("messages").Changes().Run(database.GetRethinkSession())
	if err != nil {
		log.Println("RethinkDB error:", err)
		return
	}
	defer cursor.Close()

	var change map[string]interface{}
	for cursor.Next(&change) {
		if change["new_val"] != nil {
			// Message sent to the room
			messageID := change["new_val"].(map[string]interface{})["id"].(string)
			roomID := change["new_val"].(map[string]interface{})["roomID"].(string)
			userID := change["new_val"].(map[string]interface{})["userID"].(string)
			userName := change["new_val"].(map[string]interface{})["userName"].(string)
			userAvatar := change["new_val"].(map[string]interface{})["userAvatar"].(string)
			content := change["new_val"].(map[string]interface{})["content"].(string)

			// Broadcast message to the room
			broadcastToRoom(roomID, "new_message", fiber.Map{
				"messageID":  messageID,
				"userID":     userID,
				"userName":   userName,
				"userAvatar": userAvatar,
				"content":    content,
			})
		}
	}

	if err := cursor.Err(); err != nil {
		log.Println("Cursor error:", err)
	}
}

func WatchRoomGameChanges() {
	cursor, err := rethinkdb.Table("rooms").Changes().Run(database.GetRethinkSession())
	if err != nil {
		log.Println("RethinkDB error:", err)
		return
	}
	defer cursor.Close()

	var change map[string]interface{}
	for cursor.Next(&change) {
		newVal, newExists := change["new_val"].(map[string]interface{})
		oldVal, oldExists := change["old_val"].(map[string]interface{})

		// If the room settings are updated
		if newExists && oldExists && newVal["settings"] != oldVal["settings"] {
			log.Println("Room settings updated, triggering broadcast")
			broadcastToRoom(newVal["id"].(string), "room_updated", fiber.Map{
				"settings": newVal["settings"],
			})
		}

		// If a new room is added
		if newExists && !oldExists {
			log.Println("Room added, triggering broadcast")
			broadcastRoomCountByGame()
		}

		// If a room is deleted
		if oldExists && !newExists {
			log.Println("Room deleted, checking games table")

			// Check for any remaining games in this room
			gameCursor, err := rethinkdb.Table("games").Filter(rethinkdb.Row.Field("roomId").Eq(oldVal["id"])).Changes().Run(database.GetRethinkSession())
			if err != nil {
				log.Println("RethinkDB error when listening for game changes:", err)
				continue
			}
			defer gameCursor.Close()

			var gameChange map[string]interface{}
			for gameCursor.Next(&gameChange) {
				newGameVal, newGameExists := gameChange["new_val"].(map[string]interface{})
				_, oldGameExists := gameChange["old_val"].(map[string]interface{})

				// If a game has started
				if newGameExists && newGameVal["status"] == "started" {
					log.Println("Game started, triggering broadcast")
					broadcastToRoom(oldVal["id"].(string), "game_started", fiber.Map{
						"status": "Game has started!",
					})
				}

				// If a game was deleted and no more games remain in the room
				if oldGameExists && !newGameExists {
					countCursor, err := rethinkdb.Table("games").Filter(rethinkdb.Row.Field("roomId").Eq(oldVal["id"])).Count().Run(database.GetRethinkSession())
					if err != nil {
						log.Println("RethinkDB error when counting remaining games:", err)
						continue
					}
					defer countCursor.Close()

					var remainingGames int
					countCursor.Next(&remainingGames)
					if remainingGames == 0 {
						log.Println("No more games in the room, triggering broadcast")
						broadcastToRoom(oldVal["id"].(string), "game_ended", fiber.Map{
							"status": "Game has ended!",
						})
					}
				}
			}
		}
	}

	if err := cursor.Err(); err != nil {
		log.Println("Cursor error:", err)
	}
}

func StartWebSocketServiceGameRoom() {
	// Start goroutines to watch for specific changes in the "rooms" table and trigger broadcasts
	go WatchRoomDelete()            // Watches for room additions and deletions
	go WatchRoomUserParticipation() // Watches for user participation (join/leave)
	go WatchRoomMessages()
	go WatchRoomGameChanges()

}
