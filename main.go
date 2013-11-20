package main

import (
	"fmt"
	_ "net/http"

	"github.com/robertknight/eveirc/db"
	_ "github.com/thoj/go-ircevent"
)

func main() {
	// setup HTTP server

	// setup IRC connection manager

	// setup channel store
	var store db.DataStore
	store.Init()

	servers, err := store.ListServers()
	if err != nil {
		fmt.Printf("error listing servers - %v\n", err)
	}

	for _, server := range servers {
		fmt.Printf("server %v\n", server)
	}
}
