package handler

import (
	"github.com/goccy/go-json"
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	rethink "gopkg.in/rethinkdb/rethinkdb-go.v6"
	"log"
	"norex/database"
	"time"
)

// Initialize a map to keep track of connected clients
var clients = make(map[*websocket.Conn]bool)

func AllGamesSocket(c *websocket.Conn) {
	type WebSocketMessage struct {
		Message string `json:"message"`
	}

	var (
		mt  int
		msg []byte
		err error
	)

	for {
		if mt, msg, err = c.ReadMessage(); err != nil {
			response := WebSocketMessage{Message: "Error reading message from WebSocket"}
			jsonResponse, _ := json.Marshal(response)
			c.WriteMessage(mt, jsonResponse) // Respond with error as JSON
			break
		}

		// Log the received message and send it back as a JSON response
		log.Printf("recv: %s", msg)
		response := WebSocketMessage{Message: string(msg)} // Converting received message to JSON format
		jsonResponse, _ := json.Marshal(response)

		if err = c.WriteMessage(mt, jsonResponse); err != nil {
			response := WebSocketMessage{Message: "Error writing message to WebSocket"}
			jsonResponse, _ := json.Marshal(response)
			c.WriteMessage(mt, jsonResponse) // Respond with error as JSON
			break
		}
	}
}

// Function to broadcast room count to all connected clients
func broadcastRoomCount(conn *websocket.Conn) {
	for {
		count, err := getRoomCount()
		if err != nil {
			log.Println("Failed to get room count:", err)
			return
		}

		// Broadcast to all clients
		for client := range clients {
			if err := client.WriteJSON(fiber.Map{"roomCount": count}); err != nil {
				log.Println("Failed to send message to client:", err)
				client.Close()
				delete(clients, client)
			}
		}
		time.Sleep(5 * time.Second) // Adjust the interval as needed
	}
}

// Function to get the current count of rooms
func getRoomCount() (int, error) {
	cursor, err := rethink.Table("rooms").Count().Run(database.GetRethinkSession())
	if err != nil {
		return 0, err
	}
	var count int
	if err := cursor.One(&count); err != nil {
		return 0, err
	}
	return count, nil
}
