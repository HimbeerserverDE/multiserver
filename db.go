package main

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	*sql.DB
}

// OpenSQLite3 opens and returns a SQLite3 database
func OpenSQLite3(name, initSQL string) (*DB, error) {
	os.Mkdir("storage", 0777)

	db, err := sql.Open("sqlite3", "storage/"+name)
	if err != nil {
		return nil, err
	}

	if _, err := db.Exec(initSQL); err != nil {
		db.Close()
		return nil, err
	}

	return &DB{DB: db}, nil
}

// OpenPSQL opens and returns a PostgreSQL database
func OpenPSQL(host, name, user, password string, port uint16) (*DB, error) {
	psqlconn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, name)

	db, err := sql.Open("postgres", psqlconn)
	if err != nil {
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		db.Close()
		return nil, err
	}

	return &DB{DB: db}, nil
}

// Query executes a SQL statement and stores the results
func (db *DB) QueryRow(sql string, values []interface{}, results ...interface{}) error {
	return db.DB.QueryRow(sql, values...).Scan(results...)
}
