package database

import (
	"log"

	rethink "gopkg.in/rethinkdb/rethinkdb-go.v6"
)

var session *rethink.Session

// ConnectRethinkDB establishes a connection to RethinkDB with username and password.
func ConnectRethinkDB() {
	var err error
	session, err = rethink.Connect(rethink.ConnectOpts{
		Address:  "services.frn2.chabokan.net:23188", // Change if your RethinkDB is hosted elsewhere
		Database: "norex-real-time",                  // Replace with your database name
		Username: "admin",                            // Replace with your username
		Password: "XRNnCq8fAFZ727yz",                 // Replace with your password
	})
	if err != nil {
		log.Fatalf("Failed to connect to RethinkDB: %v", err)
	} else {
		log.Println("Connected to RethinkDB")
	}
}

// GetRethinkSession returns the RethinkDB session.
func GetRethinkSession() *rethink.Session {
	return session
}
