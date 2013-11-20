package db

import (
	"bufio"
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/user"
	"path"
	"strings"

	"github.com/coopernurse/gorp"
	_ "github.com/mattn/go-sqlite3"
)

type Account struct {
	Id       int
	Email    string
	Name     string
	PassHash string
}

type Server struct {
	Id          int
	Name        string
	Host        string
	Port        int16
	Description string
}

type Channel struct {
	Id          int
	ServerId    int `db:"server_id"`
	Name        string
	Description string
	Banner      string
	IconUrl     string `db:"icon_url"`
}

type DataStore struct {
	dbmap *gorp.DbMap
}

func checkErr(err error, message string) {
	if err != nil {
		log.Fatalf("%s: %v\n", message, err)
	}
}

func schemaVersion(db *sql.DB) int {
	_, err := db.Exec("CREATE TABLE IF NOT EXISTS schema (version INT NOT NULL)")
	checkErr(err, "Failed to create schema table")

	rows, err := db.Query("SELECT version FROM schema")
	checkErr(err, "Failed to read schema version")

	for rows.Next() {
		var version int
		rows.Scan(&version)
		return version
	}

	return 0
}

func updateSchema(db *sql.DB) {
	currentSchema := schemaVersion(db)
	schemaFile, err := os.Open("db/schema.sql")
	checkErr(err, "Failed to open schema file")

	schemaScanner := bufio.NewScanner(schemaFile)
	var stmt string
	stmtSchemaVersion := 0
	for schemaScanner.Scan() {
		line := schemaScanner.Text()
		if strings.HasPrefix(line, "/**") {
			stmtSchemaVersion++
		}
		stmt += line
		if strings.HasSuffix(line, ";") {
			if stmtSchemaVersion > currentSchema {
				_, err := db.Exec(stmt)
				checkErr(err, "Failed to apply schema")
			}
			stmt = ""
		}
	}
	if stmtSchemaVersion > currentSchema {
		log.Printf("updated to schema v %d\n", stmtSchemaVersion)
	}

	_, err = db.Exec("DELETE FROM schema")
	checkErr(err, "Unable to remove current schema version")

	_, err = db.Exec(fmt.Sprintf("INSERT INTO schema VALUES (%d)", stmtSchemaVersion))
	checkErr(err, "Failed to insert updated schema version")
}

func (ds *DataStore) Init() {
	usr, _ := user.Current()
	eveDataDir := path.Join(usr.HomeDir, ".eveirc")

	err := os.MkdirAll(eveDataDir, os.ModeDir | 0777)
	checkErr(err, "Failed to create data dir")

	dbPath := path.Join(eveDataDir, "db.sqlite")
	db, err := sql.Open("sqlite3", dbPath)
	checkErr(err, "Failed to open DB")

	setupStmts := []string{"pragma journal_mode=wal",
		"pragma synchronous=normal"}
	for _, stmt := range setupStmts {
		_, err := db.Exec(stmt)
		checkErr(err, "Failed to apply stmt")
	}

	updateSchema(db)

	ds.dbmap = &gorp.DbMap{Db: db, Dialect: gorp.SqliteDialect{}}
	ds.dbmap.AddTableWithName(Account{}, "accounts").SetKeys(true /* auto-increment */, "Id")
	ds.dbmap.AddTableWithName(Server{}, "servers").SetKeys(true /* auto-increment */, "Id")
	ds.dbmap.AddTableWithName(Channel{}, "channels").SetKeys(true /* auto-increment */, "Id")
}

func (ds *DataStore) AddServer(srv *Server) error {
	return ds.dbmap.Insert(srv)
}

func (ds *DataStore) RemoveServer(srv *Server) error {
	_, err := ds.dbmap.Delete(srv)
	return err
}

func (ds *DataStore) AddChannel(ch *Channel) error {
	return ds.dbmap.Insert(ch)
}

func (ds *DataStore) RemoveChannel(ch *Channel) error {
	_, err := ds.dbmap.Delete(ch)
	return err
}

func (ds *DataStore) ListServers() ([]Server, error) {
	servers := []Server{}
	_, err := ds.dbmap.Select(&servers, "SELECT * FROM servers")
	return servers, err
}

func (ds *DataStore) ListChannels() ([]Channel, error) {
	channels := []Channel{}
	_, err := ds.dbmap.Select(&channels, "SElECT * FROM channels")
	return channels, err
}

func (ds *DataStore) AddAccount(account *Account) error {
	return ds.dbmap.Insert(account)
}

func (ds *DataStore) FindAccountByLogin(email string) (Account, error) {
	account := Account{}
	err := ds.dbmap.SelectOne(&account, "SELECT * FROM accounts WHERE email := ?", email)
	return account, err
}
