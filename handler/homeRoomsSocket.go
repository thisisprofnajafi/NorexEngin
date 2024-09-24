package handler

import (
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"gopkg.in/rethinkdb/rethinkdb-go.v6"
	"log"
	"norex/database"
)

var clients = make(map[*websocket.Conn]bool) // connected clients

// WebSocket connection handler
func HandleGameRooms(c *websocket.Conn) {
	clients[c] = true
	defer func() {
		delete(clients, c) // remove client on disconnect
		c.Close()
	}()

	// Keep listening to broadcast messages
	for {
		_, _, err := c.ReadMessage()
		if err != nil {
			log.Println("WebSocket error:", err)
			delete(clients, c)
			c.Close()
			break
		}
	}
}

// Function to broadcast room counts separated by gameName to all clients
func broadcastRoomCountByGame() {

	res, err := rethinkdb.Table("rooms").Run(database.GetRethinkSession())
	if err != nil {
		log.Println("RethinkDB error:", err)
	}
	defer res.Close()

	var rooms []struct {
		GameName string `rethinkdb:"gameName"`
	}

	// Fetch all rooms into the slice
	err = res.All(&rooms)
	if err != nil {
		log.Println("Error fetching rooms:", err)
	}

	// Manually aggregate rooms by gameName
	gameRoomCounts := make(map[string]int)
	for _, room := range rooms {
		gameRoomCounts[room.GameName]++
	}

	// Send the room counts to all connected clients
	for client := range clients {
		if err := client.WriteJSON(fiber.Map{
			"gameRoomCounts": gameRoomCounts, // e.g., {"uno": 3, "chess": 5}
		}); err != nil {
			log.Println("Error broadcasting to client:", err)
			client.Close()
			delete(clients, client)
		}
	}
}

// Function to watch for changes (like insertions) in the RethinkDB rooms table
func WatchRoomChanges() {
	cursor, err := rethinkdb.Table("rooms").Changes().Run(database.GetRethinkSession())
	if err != nil {
		log.Println("RethinkDB error:", err)
		return
	}
	defer cursor.Close()

	var change map[string]interface{}
	for cursor.Next(&change) {
		// Broadcast only when a new room is added (you can customize it for other changes if necessary)
		if change["new_val"] != nil {
			log.Println("New room added, triggering broadcast")
			broadcastRoomCountByGame()
		}
	}

	if err := cursor.Err(); err != nil {
		log.Println("Cursor error:", err)
	}
}

// Start the WebSocket and RethinkDB watchers concurrently
func StartWebSocketService() {
	// Start a goroutine to watch for changes in the "rooms" table and trigger broadcasts
	go WatchRoomChanges()
}
