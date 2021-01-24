package main

import (
	"database/sql"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

func initStorageDB() (*sql.DB, error) {
	os.Mkdir("storage", 0775)

	db, err := sql.Open("sqlite3", "storage/storage.sqlite")
	if err != nil {
		return nil, err
	}
	if db == nil {
		panic("DB is nil")
	}

	sql_table := `CREATE TABLE IF NOT EXISTS storage (
		key VARCHAR(512) NOT NULL,
		value VARCHAR(512) NOT NULL
	);
	`

	_, err = db.Exec(sql_table)
	if err != nil {
		return nil, err
	}

	return db, nil
}

func modOrAddStorageItem(db *sql.DB, key, value string) error {
	deleteStorageItem(db, key)

	sql_addStorageItem := `INSERT INTO storage (
		key,
		value
	) VALUES (
		?,
		?
	);
	`

	stmt, err := db.Prepare(sql_addStorageItem)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(key, value)
	if err != nil {
		return err
	}

	return nil
}

func readStorageItem(db *sql.DB, key string) (string, error) {
	sql_readStorageItem := `SELECT value FROM storage WHERE key = ?;`

	stmt, err := db.Prepare(sql_readStorageItem)
	if err != nil {
		return "", err
	}
	defer stmt.Close()

	rows, err := stmt.Query(key)
	if err != nil {
		return "", err
	}

	var r string

	for rows.Next() {
		err = rows.Scan(&r)
	}

	return r, nil
}

func deleteStorageItem(db *sql.DB, key string) error {
	sql_deleteStorageItem := `DELETE FROM storage WHERE key = ?;`

	stmt, err := db.Prepare(sql_deleteStorageItem)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(key)
	if err != nil {
		return err
	}

	return nil
}

// SetStorageKey sets an entry in the storage database
func SetStorageKey(key, value string) error {
	db, err := initStorageDB()
	if err != nil {
		return err
	}
	defer db.Close()

	if value == "" {
		return deleteStorageItem(db, key)
	}

	return modOrAddStorageItem(db, key, value)
}

// GetStorageKey gets an entry in the storage database
func GetStorageKey(key string) (string, error) {
	db, err := initStorageDB()
	if err != nil {
		return "", err
	}
	defer db.Close()

	return readStorageItem(db, key)
}
