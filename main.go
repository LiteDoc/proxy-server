package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/gocql/gocql"
	"github.com/gorilla/mux"

	"github.com/gorilla/sessions"
	"github.com/rogerwangcs/proxy-server/Cassandra"
)

type JSONResponse struct {
	Status string
	Code   int
}

type ClientWriter struct {
	ID   string
	Name string
}

func heartbeat(w http.ResponseWriter, r *http.Request, serverID int) {
	message := fmt.Sprintf("Proxy Server %d is up and running.", serverID+4000)
	json.NewEncoder(w).Encode(message)
	// w.Write([]byte(message))
}

// type response struct {
// 	Status string
// 	res []map[string]interface{}
// }

// ConnectHandler : start new session
func ConnectHandler(w http.ResponseWriter, r *http.Request, serverID int, store *sessions.CookieStore) {
	session, err := store.Get(r, "session-name")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Print(session)
	// Set some session values.
	session.Values["foo"] = "bar"
	session.Values[42] = 43
	// Save it before we write to the response/return from the handler.
	err = session.Save(r, w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(session)
}

// ReadHandler : read
func ReadHandler(w http.ResponseWriter, r *http.Request, serverID int, cass *gocql.Session) {
	log.Print(fmt.Sprintf("Read on server %d. \n", serverID+4000))

	q := cass.Query(`SELECT * from testtable`)
	rows, ok := q.Iter().SliceMap()
	if ok != nil {
		panic(fmt.Sprintf("%s\n", ok.Error()))
	}
	for _, row := range rows {
		for _, val := range row {
			fmt.Printf(" %s ", val)
		}
		fmt.Printf("\n")
	}

	json.NewEncoder(w).Encode(rows)
}

// WriteHandler : write
func WriteHandler(w http.ResponseWriter, r *http.Request, serverID int, cass *gocql.Session) {
	log.Print(fmt.Sprintf("Write on server %d. \n", serverID+4000))

	err := cass.Query(`INSERT INTO testtable (id, field0, tag) VALUES (5, 'yes work', '5')`).Exec()
	if err != nil {
		log.Fatal(err)
	}

	json.NewEncoder(w).Encode(JSONResponse{Status: "Write Success", Code: 200})
}

func serverThread(serverID int, portShift int, wg *sync.WaitGroup) {
	defer wg.Done() // finish waitgroup after thread returns

	// Initialize Session Store
	var store = sessions.NewCookieStore([]byte("temp session key"))

	// Initialize Router
	router := mux.NewRouter().StrictSlash(true)

	// Routes
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		heartbeat(w, r, serverID)
	})
	router.HandleFunc("/connect", func(w http.ResponseWriter, r *http.Request) {
		ConnectHandler(w, r, serverID, store)
	})
	router.HandleFunc("/read", func(w http.ResponseWriter, r *http.Request) {
		ReadHandler(w, r, serverID, Cassandra.Session)
	})
	router.HandleFunc("/write", func(w http.ResponseWriter, r *http.Request) {
		WriteHandler(w, r, serverID, Cassandra.Session)
	})

	// Listen Server
	log.Print(fmt.Sprintf("Listening on port %d: \n", serverID+portShift))
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", serverID+portShift), router))

}

func main() {

	portShift := 4000

	// Start server threads
	var wg sync.WaitGroup
	for i := 1; i <= 5; i++ {
		wg.Add(1)
		go serverThread(i, portShift, &wg)
	}
	wg.Wait()

}
