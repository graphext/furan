package datalayer_test

import (
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/dollarshaveclub/furan/lib/config"
	"github.com/dollarshaveclub/furan/lib/db"
	"github.com/dollarshaveclub/go-lib/cassandra"
	"github.com/gocql/gocql"
)

const (
	testKeyspace = "furan_test"
)

var tn = os.Getenv("SCYLLA_TEST_NODE")
var ts *gocql.Session
var dbConfig config.DBconfig

var testDBTimeout = 5 * time.Minute

func setupTestDB() {
	// create keyspace
	c := gocql.NewCluster(tn)
	c.ProtoVersion = 3
	c.NumConns = 20
	c.SocketKeepalive = testDBTimeout
	c.Timeout = testDBTimeout
	c.MaxWaitSchemaAgreement = testDBTimeout
	log.Printf("creating keyspace session...\n")
	s, err := c.CreateSession()
	if err != nil {
		log.Fatalf("error creating keyspace session: %v", err)
	}
	defer s.Close()
	log.Printf("creating keyspace...\n")
	err = s.Query(fmt.Sprintf("CREATE KEYSPACE IF NOT EXISTS %v WITH REPLICATION = {'class': 'SimpleStrategy', 'replication_factor': 1};", testKeyspace)).Exec()
	if err != nil {
		log.Fatalf("error creating keyspace: %v", err)
	}
	time.Sleep(1)

	// DB setup
	c.Keyspace = testKeyspace
	dbConfig.Cluster = c
	dbConfig.Nodes = []string{tn}
	dbConfig.Keyspace = testKeyspace
	log.Printf("creating types...\n")
	err = cassandra.CreateRequiredTypes(dbConfig.Cluster, db.RequiredUDTs)
	if err != nil {
		log.Fatalf("error creating UDTs: %v", err)
	}
	log.Printf("creating tables...\n")
	err = cassandra.CreateRequiredTables(dbConfig.Cluster, db.RequiredTables)
	if err != nil {
		log.Fatalf("error creating tables: %v", err)
	}
	ts, err = dbConfig.Cluster.CreateSession()
	if err != nil {
		log.Fatalf("error getting session: %v", err)
	}
}

func teardownTestDB() {
	q := fmt.Sprintf("DROP KEYSPACE %v;", testKeyspace)
	err := ts.Query(q).Exec()
	if err != nil {
		log.Fatalf("error dropping keyspace: %v", err)
	}
}

func TestMain(m *testing.M) {
	var exit int
	defer func() {
		os.Exit(exit)
	}()
	if tn == "" {
		fmt.Println("SCYLLA_TEST_NODE must be defined: <host/ip>:<port>")
		os.Exit(1)
	} else {
		setupTestDB()
		defer teardownTestDB()
	}
	exit = m.Run()
}
