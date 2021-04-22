package main

import (
	"database/sql"
	"fmt"
	"os"
	"regexp"

	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

const (
	DBTypeSQLite3 = iota
	DBTypePSQL
)

type DB struct {
	*sql.DB
	dbType int
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

	return &DB{DB: db, dbType: DBTypeSQLite3}, nil
}

// OpenPSQL opens and returns a PostgreSQL database
func OpenPSQL(host, name, user, password, initSQL string, port int) (*DB, error) {
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

	if _, err := db.Exec(initSQL); err != nil {
		db.Close()
		return nil, err
	}

	return &DB{DB: db, dbType: DBTypePSQL}, nil
}

// Type returns the type of database that is being interacted with
func (db *DB) Type() int { return db.dbType }

// Exec executes a SQL statement
func (db *DB) Exec(sql string, values ...interface{}) (sql.Result, error) {
	if db.Type() == DBTypeSQLite3 {
		r, err := regexp.Compile("\\$+[0-9]")
		if err != nil {
			return nil, err
		}

		sql = r.ReplaceAllString(sql, "?")
	}
	return db.DB.Exec(sql, values...)
}

// QueryRow executes a SQL statement and stores the results
func (db *DB) QueryRow(sql string, values ...interface{}) *sql.Row {
	if db.Type() == DBTypeSQLite3 {
		r, err := regexp.Compile("\\$+[0-9]")
		if err != nil {
			return nil
		}

		sql = r.ReplaceAllString(sql, "?")
	}
	return db.DB.QueryRow(sql, values...)
}
