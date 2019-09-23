package storage

import (
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // Register DB driver for PostgreSQL
)

// DB is exported for use in sub-packages
var DB *sqlx.DB

// InitDB is called from main(), to set up DB and check for connectivity
func InitDB(connString string) error {
	driverName := "postgres"
	var err error
	if DB, err = sqlx.Open(driverName, connString); err == nil {
		if err := DB.Ping(); err == nil {
			go _onReady() // launch functions dependant on DBs readiness
		} else {
			return err // Error at DB.Ping() call
		}
	} else {
		return err // Error at sqlx.Open() call
	}
	return nil
}

var onReady = make(chan func())

// OnReady allows sub-packages to queue their own init() once we have a DB up
func OnReady(initdb func()) {
	go func() { // goroutines launched here will stay blocked till _onReady fires
		onReady <- initdb
	}()
}

func _onReady() {
	for fn := range onReady {
		go fn() // launch functions dependant on DBs readiness
	}
}
