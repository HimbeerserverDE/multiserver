package multiserver

import (
	"database/sql"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

// initPluginStorageDB opens plugin_storage.sqlite
// and creates the required tables if they don't exist
// It returns said database
func initPluginStorageDB() (*sql.DB, error) {
	os.Mkdir("storage", 0775)

	db, err := sql.Open("sqlite3", "storage/plugin_storage.sqlite")
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

// modOrAddPluginStorageItem updates a plugin storage DB entry
// and inserts it if it doesn't exist
func modOrAddPluginStorageItem(db *sql.DB, key, value string) error {
	deletePluginStorageItem(db, key)

	sql_addPluginStorageItem := `INSERT INTO storage (
		key,
		value
	) VALUES (
		?,
		?
	);
	`

	stmt, err := db.Prepare(sql_addPluginStorageItem)
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

// readPluginStorageItem selects and reads a plugin storage DB entry
func readPluginStorageItem(db *sql.DB, key string) (string, error) {
	sql_readPluginStorageItem := `SELECT value FROM storage WHERE key = ?;`

	stmt, err := db.Prepare(sql_readPluginStorageItem)
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

// deletePluginStorageItem deletes a plugin storage DB entry
func deletePluginStorageItem(db *sql.DB, key string) error {
	sql_deletePluginStorageItem := `DELETE FROM storage WHERE key = ?;`

	stmt, err := db.Prepare(sql_deletePluginStorageItem)
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
