package handler

import (
	"context"
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"gopkg.in/rethinkdb/rethinkdb-go.v6"
	"log"
	"math/rand"
	"norex/database" // Adjust the import path according to your project structure
	"norex/models"
	"strings"
)

var gameClients = make(map[string]map[*websocket.Conn]bool) // connected clients grouped by game

func generateRoomID() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 9)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

func CreateRoom(c *fiber.Ctx) error {
	var room struct {
		RoomID       string `json:"roomId"`
		GameName     string `json:"gameName"`
		IsLocked     bool   `json:"isLocked"`
		RoomPassword string `json:"roomPassword"`
		VoiceChatOn  bool   `json:"voiceChatOn"`
		TextChatOn   bool   `json:"textChatOn"`
		MinLevel     int    `json:"minLevel"`
		UserEmail    string `json:"userEmail"`
		Avatar       string `json:"avatar"`
		Name         string `json:"name"`
		Capacity     int    `json:"capacity"`
	}

	// Generate a unique RoomID
	room.RoomID = generateRoomID()

	// Get the user's email from c.Locals
	email := c.Locals("email").(string)

	// Fetch the user ID from MongoDB using the email
	var user models.User
	if err := database.GetCollection("users").FindOne(context.TODO(), bson.M{"email": email}).Decode(&user); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to find user"})
	}
	room.UserEmail = email
	room.Avatar = user.Avatar
	room.Name = user.Name

	// Parse the JSON body
	if err := c.BodyParser(&room); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	// Get the RethinkDB session
	session := database.GetRethinkSession()

	// Insert the room into RethinkDB
	_, err := rethinkdb.Table("rooms").Insert(room).Run(session)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create room " + err.Error()})
	}

	return c.JSON(fiber.Map{"message": "Room created successfully", "roomId": room.RoomID, "gameName": room.GameName})
}

func GetGameRooms(c *fiber.Ctx) error {
	gameName := c.Params("game_name")

	// Always fetch with 0 offset and max 50 limit
	const offset = 0
	const limit = 50

	// Query the rooms based on gameName, with fixed offset and limit
	cursor, err := rethinkdb.Table("rooms").
		Filter(rethinkdb.Row.Field("GameName").Eq(gameName)).
		Without("RoomPassword"). // Exclude specific fields
		Skip(offset).
		Limit(limit).
		Run(database.GetRethinkSession())
	if err != nil {
		log.Println("RethinkDB error:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Error fetching rooms"})
	}
	defer cursor.Close()

	var rooms []map[string]interface{}
	err = cursor.All(&rooms)
	if err != nil {
		log.Println("Error fetching rooms:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Error fetching rooms"})
	}

	return c.JSON(rooms)
}

func HandleNewGameRoom(c *websocket.Conn) {
	// Extract game name from the URL (e.g., /game/uno)
	gameName := c.Params("game_name")
	gameName = strings.ToLower(gameName) // Normalize game name

	// Initialize map for the game if not already done
	if gameClients[gameName] == nil {
		gameClients[gameName] = make(map[*websocket.Conn]bool)
	}

	// Add the client to the game-specific map
	gameClients[gameName][c] = true
	defer func() {
		delete(gameClients[gameName], c) // remove client on disconnect
		c.Close()
	}()

	// Keep listening to broadcast messages
	for {
		_, _, err := c.ReadMessage()
		if err != nil {
			log.Println("WebSocket error:", err)
			delete(gameClients[gameName], c)
			c.Close()
			break
		}
	}
}

// Broadcast new room addition
func broadcastNewRoomByGame(gameName string, newRoom map[string]interface{}) {
	for client := range gameClients[gameName] {
		if err := client.WriteJSON(fiber.Map{
			"eventType": "added", // Event type: room added
			"newRoom":   newRoom, // Full room details
		}); err != nil {
			log.Println("Error broadcasting to client:", err)
			client.Close()
			delete(gameClients[gameName], client)
		}
	}
}

// Broadcast room deletion
func broadcastDeletedRoom(gameName string, roomDetails map[string]interface{}) {
	for client := range gameClients[gameName] {
		if err := client.WriteJSON(fiber.Map{
			"eventType": "deleted", // Event type: room deleted
			"room":      roomDetails,
		}); err != nil {
			log.Println("Error broadcasting to client:", err)
			client.Close()
			delete(gameClients[gameName], client)
		}
	}
}

// Broadcast room updates (changes)
func broadcastRoomChange(gameName string, roomDetails map[string]interface{}) {
	for client := range gameClients[gameName] {
		if err := client.WriteJSON(fiber.Map{
			"eventType": "updated", // Event type: room updated
			"room":      roomDetails,
		}); err != nil {
			log.Println("Error broadcasting to client:", err)
			client.Close()
			delete(gameClients[gameName], client)
		}
	}
}
func WatchRoomGameAddOrDelete() {
	cursor, err := rethinkdb.Table("rooms").Changes().Run(database.GetRethinkSession())
	if err != nil {
		log.Println("RethinkDB error:", err)
		return
	}
	defer cursor.Close()

	var change map[string]interface{}
	for cursor.Next(&change) {
		// If new value exists but old value is nil, a new room was added
		if newVal, newExists := change["new_val"].(map[string]interface{}); newExists && change["old_val"] == nil {
			if gameName, exists := newVal["GameName"].(string); exists {
				log.Printf("New room added for game: %s", gameName)
				broadcastNewRoomByGame(strings.ToLower(gameName), newVal)
			}
		}

		// If old value exists but new value is nil, a room was deleted
		if oldVal, oldExists := change["old_val"].(map[string]interface{}); oldExists && change["new_val"] == nil {
			if gameName, exists := oldVal["GameName"].(string); exists {
				log.Printf("Room deleted for game: %s", gameName)
				broadcastDeletedRoom(strings.ToLower(gameName), oldVal)
			}
		}
	}

	if err := cursor.Err(); err != nil {
		log.Println("Cursor error:", err)
	}
}

func WatchGameRoomChanges() {
	cursor, err := rethinkdb.Table("rooms").Changes().Run(database.GetRethinkSession())
	if err != nil {
		log.Println("RethinkDB error:", err)
		return
	}
	defer cursor.Close()

	var change map[string]interface{}
	for cursor.Next(&change) {
		// If both new_val and old_val exist, it means a room was updated
		if newVal, newExists := change["new_val"].(map[string]interface{}); newExists && change["old_val"] != nil {
			if gameName, exists := newVal["GameName"].(string); exists {
				log.Printf("Room updated for game: %s", gameName)
				broadcastRoomChange(strings.ToLower(gameName), newVal) // Broadcast room update
			}
		}
	}

	if err := cursor.Err(); err != nil {
		log.Println("Cursor error:", err)
	}
}

func StartWebSocketServiceNewGameInfo() {
	// Start goroutines to watch for specific changes in the "rooms" table and trigger broadcasts
	go WatchRoomGameAddOrDelete() // Watches for room additions and deletions
	go WatchGameRoomChanges()     // Watches for room updates/changes
}

type RoomUpdate struct {
	GameName     *string `json:"gameName,omitempty"`
	IsLocked     *bool   `json:"isLocked,omitempty"`
	RoomPassword *string `json:"roomPassword,omitempty"`
	VoiceChatOn  *bool   `json:"voiceChatOn,omitempty"`
	TextChatOn   *bool   `json:"textChatOn,omitempty"`
	MinLevel     *int    `json:"minLevel,omitempty"`
	Capacity     *int    `json:"capacity,omitempty"`
}

func EditRoom(c *fiber.Ctx) error {
	roomID := c.Params("id")                // Get the room ID from the URL
	userEmail := c.Locals("email").(string) // Get the authenticated user's email

	// Fetch the room by ID
	cursor, err := rethinkdb.Table("rooms").Get(roomID).Run(database.GetRethinkSession())
	if err != nil {
		log.Println("Error fetching room:", err)
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Room not found"})
	}
	defer cursor.Close()

	// Create a variable to hold the room data
	var room map[string]interface{}

	// Check if a room was found by attempting to read it
	if err := cursor.One(&room); err != nil {
		log.Println("Error reading room:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Error reading room"})
	}

	// Check if the user is the owner of the room
	if ownerID, exists := room["UserEmail"].(string); !exists || ownerID != userEmail {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "You are not authorized to edit this room"})
	}

	// Parse the request body for the updated room data
	var updatedRoomData RoomUpdate
	if err := c.BodyParser(&updatedRoomData); err != nil {
		log.Println("Error parsing request body:", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request data"})
	}

	// Convert struct to a map and remove nil fields for partial updates
	updateMap := make(map[string]interface{})
	if updatedRoomData.GameName != nil {
		updateMap["gameName"] = *updatedRoomData.GameName
	}
	if updatedRoomData.IsLocked != nil {
		updateMap["isLocked"] = *updatedRoomData.IsLocked
	}
	if updatedRoomData.RoomPassword != nil {
		updateMap["roomPassword"] = *updatedRoomData.RoomPassword
	}
	if updatedRoomData.VoiceChatOn != nil {
		updateMap["voiceChatOn"] = *updatedRoomData.VoiceChatOn
	}
	if updatedRoomData.TextChatOn != nil {
		updateMap["textChatOn"] = *updatedRoomData.TextChatOn
	}
	if updatedRoomData.MinLevel != nil {
		updateMap["minLevel"] = *updatedRoomData.MinLevel
	}
	if updatedRoomData.Capacity != nil {
		updateMap["capacity"] = *updatedRoomData.Capacity
	}

	// Ensure there's something to update
	if len(updateMap) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "No fields to update"})
	}

	// Update the room in the database
	_, err = rethinkdb.Table("rooms").Get(roomID).Update(updateMap).RunWrite(database.GetRethinkSession())
	if err != nil {
		log.Println("Error updating room:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Error updating room"})
	}

	return c.JSON(fiber.Map{"message": "Room updated successfully", "room": updateMap})
}
