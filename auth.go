package multiserver

import (
	"database/sql"
	"encoding/base64"
	_ "github.com/mattn/go-sqlite3"
	"os"
	"strings"
)

const (
	AuthMechSRP      = 0x00000002
	AuthMechFirstSRP = 0x00000004
)

// encodeVerifierAndSalt encodes SRP verifier and salt into DB-ready string
func encodeVerifierAndSalt(s, v []byte) string {
	return base64.StdEncoding.EncodeToString(s) + "#" + base64.StdEncoding.EncodeToString(v)
}

// decodeVerifierAndSalt decodes DB-ready string into SRP verifier and salt
func decodeVerifierAndSalt(src string) ([]byte, []byte, error) {
	sString := strings.Split(src, "#")[0]
	vString := strings.Split(src, "#")[1]

	s, err := base64.StdEncoding.DecodeString(sString)
	if err != nil {
		return nil, nil, err
	}

	v, err := base64.StdEncoding.DecodeString(vString)
	if err != nil {
		return nil, nil, err
	}

	return s, v, nil
}

// initAuthDB opens auth.sqlite and creates the required tables
// if they don't exist
// It returns said database
func initAuthDB() (*sql.DB, error) {
	os.Mkdir("storage", 0775)

	db, err := sql.Open("sqlite3", "storage/auth.sqlite")
	if err != nil {
		return nil, err
	}
	if db == nil {
		panic("DB is nil")
	}

	sql_table := `CREATE TABLE IF NOT EXISTS auth (
		name VARCHAR(32) NOT NULL,
		password VARCHAR(512) NOT NULL
	);
	CREATE TABLE IF NOT EXISTS privileges (
		name VARCHAR(32) NOT NULL,
		privileges VARCHAR(1024)
	);
	`

	_, err = db.Exec(sql_table)
	if err != nil {
		return nil, err
	}

	return db, nil
}

// addAuthItem inserts an auth DB entry
func addAuthItem(db *sql.DB, name, password string) error {
	sql_addAuthItem := `INSERT INTO auth (
		name,
		password
	) VALUES (
		?,
		?
	);
	`

	stmt, err := db.Prepare(sql_addAuthItem)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(name, password)
	if err != nil {
		return err
	}

	return nil
}

// modAuthItem updates an auth DB entry
func modAuthItem(db *sql.DB, name, password string) error {
	sql_modAuthItem := `UPDATE auth SET password = ? WHERE name = ?;`

	stmt, err := db.Prepare(sql_modAuthItem)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(password, name)
	if err != nil {
		return err
	}

	return nil
}

// readAuthItem selects and reads an auth DB entry
func readAuthItem(db *sql.DB, name string) (string, error) {
	sql_readAuthItem := `SELECT password FROM auth WHERE name = ?;`

	stmt, err := db.Prepare(sql_readAuthItem)
	if err != nil {
		return "", err
	}
	defer stmt.Close()

	rows, err := stmt.Query(name)
	if err != nil {
		return "", err
	}

	var r string

	for rows.Next() {
		err = rows.Scan(&r)
	}

	return r, nil
}
