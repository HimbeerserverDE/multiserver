package multiserver

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"os"
)

// InitStorageDB opens_storage.sqlite
// and creates the required tables if they don't exist
// It returns said database
func InitStorageDB() (*sql.DB, error) {
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

// ModOrAddStorageItem updates a storage DB entry
// and inserts it if it doesn't exist
func ModOrAddStorageItem(db *sql.DB, key, value string) error {
	DeleteStorageItem(db, key)

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

// ReadStorageItem selects and reads a storage DB entry
func ReadStorageItem(db *sql.DB, key string) (string, error) {
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

// DeleteStorageItem deletes a storage DB entry
func DeleteStorageItem(db *sql.DB, key string) error {
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
