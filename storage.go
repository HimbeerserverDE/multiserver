package main

import (
	"database/sql"
	"errors"

	_ "github.com/mattn/go-sqlite3"
)

func storageDB() (*DB, error) {
	return OpenSQLite3("storage.sqlite", `CREATE TABLE IF NOT EXISTS storage (
	key VARCHAR(512) PRIMARY KEY NOT NULL,
	value VARCHAR(512) NOT NULL
);`)
}

// StorageKey returns an entry from the storage database
func StorageKey(key string) (string, error) {
	db, err := storageDB()
	if err != nil {
		return "", err
	}
	defer db.Close()

	var r string
	err = db.QueryRow(`SELECT value FROM storage WHERE key = ?;`, key).Scan(&r)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return "", err
	}

	return r, nil
}

// SetStorageKey sets an entry in the storage database
func SetStorageKey(key, value string) error {
	db, err := storageDB()
	if err != nil {
		return err
	}
	defer db.Close()

	if value == "" {
		_, err = db.Exec(`DELETE FROM storage WHERE key = ?;`)
	} else {
		_, err = db.Exec(`REPLACE INTO storage (
			key,
			value
		) VALUES (
			?,
			?
		);`, key, value)
	}
	return err
}
