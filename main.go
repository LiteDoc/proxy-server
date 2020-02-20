package main

import (
	"net/http"
	"log"
	"strconv"
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/rogerwangcs/proxy-server/Cassandra"
	// "github.com/gocql/gocql"
)

type heartbeatResponse struct {
  Status string
  Code int
}

func heartbeat(w http.ResponseWriter, r *http.Request) {
	log.Print("get called")
  json.NewEncoder(w).Encode(heartbeatResponse{Status: "OK", Code: 200})
}

func WriteHandler(w http.ResponseWriter, r *http.Request) {
	log.Print("write called")
  json.NewEncoder(w).Encode(heartbeatResponse{Status: "OK", Code: 200})
}

func main() {

	CassandraSession := Cassandra.Session
	defer CassandraSession.Close()

	if err := CassandraSession.Query(`INSERT INTO testtable (id, field0, tag) VALUES (5, 'asd', '5')`).Exec(); err != nil {
	log.Fatal(err)
}


	port := 4001
  router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/", heartbeat)
	router.HandleFunc("/write", WriteHandler)


	log.Print("Listening on port " + strconv.Itoa(port) + ": \n")
  log.Fatal(http.ListenAndServe(":4001", router))
}