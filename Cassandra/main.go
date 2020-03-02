package Cassandra

import (
	"fmt"
	"os"

	"github.com/gocql/gocql"
	"github.com/joho/godotenv"
)

// Session : Cassandra session
var Session *gocql.Session

func init() {
	godotenv.Load()
	var err error

	cassServerIP, exists := os.LookupEnv("CASS_SERVER")
	if exists {
		cluster := gocql.NewCluster(cassServerIP)
		cluster.Keyspace = "testspace"
		Session, err = cluster.CreateSession()
		fmt.Println("cassandra init done")
	}

	if err != nil {
		panic(err)
	}
}
