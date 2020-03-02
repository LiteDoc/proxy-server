package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/joho/godotenv"
	"github.com/rs/cors"
)

func proxyServer(serverID int, wg *sync.WaitGroup) {
	defer wg.Done() // finish waitgroup after thread returns

	// Initialize connected user map
	connectedUsers := make(map[string]string)

	// Initialize cookie store
	store := sessions.NewCookieStore([]byte("temp session key"))
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   60 * 60 * 1,
		HttpOnly: true,
	}

	// Initialize Router
	router := mux.NewRouter().StrictSlash(true)
	cors.Default().Handler(router)
	AddRoutes(router, serverID, store, connectedUsers)
	cors.Default().Handler(router)

	// Start Server
	log.Print(fmt.Sprintf("Listening on port %d. \n", serverID))
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", serverID), router))

}

func main() {
	godotenv.Load()
	portShift := 4001 // first port number
	// Start server threads
	var wg sync.WaitGroup
	for i := 0; i < 3; i++ {
		wg.Add(1)
		serverID := portShift + i
		go proxyServer(serverID, &wg)
	}
	wg.Wait()

}
