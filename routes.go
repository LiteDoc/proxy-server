package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"

	"github.com/gocql/gocql"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/rogerwangcs/proxy-server/Cassandra"
)

func heartbeat(w http.ResponseWriter, r *http.Request, serverID int) {
	message := fmt.Sprintf("Proxy Server %d is up and running.", serverID)
	json.NewEncoder(w).Encode(message)
	// w.Write([]byte(message))
}

func enableCors(w *http.ResponseWriter) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
	(*w).Header().Set("Access-Control-Allow-Credentials", "true")
	(*w).Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	(*w).Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
}

// ConnectHandler : start new session
func connectHandler(w http.ResponseWriter, r *http.Request, serverID int, store *sessions.CookieStore, connectedUsers map[string]string) {

	// Get user name
	var q = r.URL.Query()
	var clientName string = q.Get("name")

	// Retrieve user's session
	session, err := store.Get(r, "session-name")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// check if client already has a session on current proxy
	if session.Values["proxy"] == serverID {
		log.Print(fmt.Sprintf("%s reconnected on server %d", session.Values["name"], serverID))
		json.NewEncoder(w).Encode(JSONResponse{Status: "Reconnected", Code: 200})
		return
	}

	// check if another client is connected to current proxy
	if len(connectedUsers) > 0 {
		log.Print("Client will switch to another server")
		json.NewEncoder(w).Encode(
			JSONResponse{Status: "connect to a different proxy", Code: 403})
		return
	}

	// create session for current user and track user in current proxy
	session.Values["name"] = clientName
	session.Values["proxy"] = serverID
	connectedUsers[clientName] = clientName
	log.Print(fmt.Sprintf("%s connected on server %d", clientName, serverID))
	log.Print(connectedUsers)

	err = session.Save(r, w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(JSONResponse{Status: "Connected", Code: 200})
}

func isOwner(clientName string, clientRegisterID string) string {
	// create query and get response
	var clientQuery string = BuildQuery("http://localhost:4000/readLock",
		[]string{"name", clientName}, []string{"registerID", clientRegisterID})
	resp, err := http.Get(clientQuery)
	if err != nil {
		log.Fatalln(err)
		return ""
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	return string(body)
}

// ReadHandler : read
func readHandler(w http.ResponseWriter, r *http.Request, serverID int, cass *gocql.Session) {

	log.Print(fmt.Sprintf("Read on server %d. \n", serverID))

	q := cass.Query(`SELECT * from registers`)
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
func writeHandler(w http.ResponseWriter, r *http.Request, serverID int, cass *gocql.Session) {
	// Get query params
	var q = r.URL.Query()
	var clientName string = q.Get("name")
	var clientRegisterID string = q.Get("registerID")
	isClientOwner, err := strconv.ParseBool(isOwner(clientName, clientRegisterID))
	if err != nil {
		log.Fatalln(err)
		return
	}
	if !isClientOwner {
		json.NewEncoder(w).Encode(JSONResponse{Status: "Write Failed", Code: 403})
		return
	}

	// get request body
	buf := new(bytes.Buffer)
	buf.ReadFrom(r.Body)
	reqBody := buf.String()

	err = cass.Query("UPDATE registers SET field0=? WHERE y_id=?", reqBody, clientRegisterID).Exec()
	if err != nil {
		log.Fatal(err)
		return
	}
	json.NewEncoder(w).Encode(JSONResponse{Status: "Write Success", Code: 200})
	return
}

// readLocksHandler : return register locks status
func readLocksHandler(w http.ResponseWriter, r *http.Request) {
	log.Print(fmt.Sprintf("Read locks. \n"))

	// create query and get response
	var clientQuery string = "http://localhost:4000/readLocks"
	resp, err := http.Get(clientQuery)
	if err != nil {
		log.Fatalln(err)
		return
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	// convert response back to a map and then encode and send
	var result map[string]interface{}
	err = json.Unmarshal(body, &result)
	json.NewEncoder(w).Encode(result)
}

// lockHandler : request register lock to LE
func lockHandler(w http.ResponseWriter, r *http.Request) {
	// Get query params
	var q = r.URL.Query()
	var clientName string = q.Get("name")
	var clientRegisterID string = q.Get("registerID")

	log.Print(clientName)
	log.Print(clientRegisterID)

	// create query and get response
	var clientQuery string = BuildQuery("http://localhost:4000/lock",
		[]string{"name", clientName}, []string{"registerID", clientRegisterID})
	resp, err := http.Get(clientQuery)
	if err != nil {
		log.Fatalln(err)
		return
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	// convert body to json and pass to client
	resBody := JSONResponse{}
	err = json.Unmarshal(body, &resBody)
	if err != nil {
		log.Fatalln(err)
		return
	}
	json.NewEncoder(w).Encode(resBody)
}

// unlockHandler : request register unlock to LE
func unlockHandler(w http.ResponseWriter, r *http.Request) {
	// Get query params
	var q = r.URL.Query()
	var clientName string = q.Get("name")
	var clientRegisterID string = q.Get("registerID")

	// create query and get response
	var clientQuery string = BuildQuery("http://localhost:4000/unlock",
		[]string{"name", clientName}, []string{"registerID", clientRegisterID})
	resp, err := http.Get(clientQuery)
	if err != nil {
		log.Fatalln(err)
		return
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	// convert body to json and pass to client
	resBody := JSONResponse{}
	err = json.Unmarshal(body, &resBody)
	if err != nil {
		log.Fatalln(err)
		return
	}
	json.NewEncoder(w).Encode(resBody)
}

// AddRoutes : attach route handlers to given mux router
func AddRoutes(router *mux.Router, serverID int, store *sessions.CookieStore, connectedUsers map[string]string) {
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		enableCors(&w)
		heartbeat(w, r, serverID)
	})
	router.HandleFunc("/connect", func(w http.ResponseWriter, r *http.Request) {
		enableCors(&w)
		connectHandler(w, r, serverID, store, connectedUsers)
	})
	router.HandleFunc("/read", func(w http.ResponseWriter, r *http.Request) {
		enableCors(&w)
		readHandler(w, r, serverID, Cassandra.Session)
	})
	router.HandleFunc("/write", func(w http.ResponseWriter, r *http.Request) {
		enableCors(&w)
		writeHandler(w, r, serverID, Cassandra.Session)
	})
	router.HandleFunc("/readLocks", func(w http.ResponseWriter, r *http.Request) {
		enableCors(&w)
		readLocksHandler(w, r)
	})
	router.HandleFunc("/lock", func(w http.ResponseWriter, r *http.Request) {
		enableCors(&w)
		lockHandler(w, r)
	})
	router.HandleFunc("/unlock", func(w http.ResponseWriter, r *http.Request) {
		enableCors(&w)
		unlockHandler(w, r)
	})
}
