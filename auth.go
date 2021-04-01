package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"log"
	"os"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

const (
	_ = 1 << iota
	AuthMechSRP
	AuthMechFirstSRP
)

var passPhrase []byte

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
	os.Mkdir("storage", 0777)

	db, err := sql.Open("sqlite3", "storage/auth.sqlite")
	if err != nil {
		return nil, err
	}

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS auth (
			name VARCHAR(32) NOT NULL,
			password VARCHAR(512) NOT NULL
		);
		CREATE TABLE IF NOT EXISTS privileges (
			name VARCHAR(32) NOT NULL,
			privileges VARCHAR(1024)
		);
		CREATE TABLE IF NOT EXISTS ban (
			addr VARCHAR(39) NOT NULL,
			name VARCHAR(32) NOT NULL
		);
	`); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

// addAuthItem inserts an auth DB entry
func addAuthItem(db *sql.DB, name, password string) error {
	_, err := db.Exec(`INSERT INTO auth (
		name,
		password
	) VALUES (
		?,
		?
	);`, name, password)
	return err
}

// modAuthItem updates an auth DB entry
func modAuthItem(db *sql.DB, name, password string) error {
	_, err := db.Exec(`UPDATE auth SET password = ? WHERE name = ?;`, password, name)
	return err
}

// readAuthItem selects and reads an auth DB entry
func readAuthItem(db *sql.DB, name string) (string, error) {
	var r string
	err := db.QueryRow(`SELECT password FROM auth WHERE name = ?;`, name).Scan(&r)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return "", err
	}

	return r, nil
}

func init() {
	pwd, err := StorageKey("auth:passphrase")
	if err != nil {
		log.Fatal(err)
	}

	if pwd == "" {
		passPhrase = make([]byte, 16)
		_, err := rand.Read(passPhrase)
		if err != nil {
			log.Fatal(err)
		}

		// Save the passphrase for future use
		// This passphrase should not be changed unless you delete
		// the auth databases on the minetest servers
		err = SetStorageKey("auth:passphrase", string(passPhrase))
		if err != nil {
			log.Fatal(err)
		}
	} else {
		passPhrase = []byte(pwd)
	}
}
