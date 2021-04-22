package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"log"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

const (
	_ = 1 << iota
	AuthMechSRP
	AuthMechFirstSRP
)

var passPhrase []byte

func encodeVerifierAndSalt(s, v []byte) string {
	return base64.StdEncoding.EncodeToString(s) + "#" + base64.StdEncoding.EncodeToString(v)
}

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

func authDB() (*DB, error) {
	sqlite3 := func() (*DB, error) {
		return OpenSQLite3("auth.sqlite", `CREATE TABLE IF NOT EXISTS auth (
	name VARCHAR(32) PRIMARY KEY NOT NULL,
	password VARCHAR(512) NOT NULL
);
CREATE TABLE IF NOT EXISTS privileges (
	name VARCHAR(32) PRIMARY KEY NOT NULL,
	privileges VARCHAR(1024)
);
CREATE TABLE IF NOT EXISTS ban (
	addr VARCHAR(39) PRIMARY KEY NOT NULL,
	name VARCHAR(32) NOT NULL
);`)
	}

	psql := func(host, name, user, password string, port int) (*DB, error) {
		return OpenPSQL(host, name, user, password, `CREATE TABLE IF NOT EXISTS auth (
	name VARCHAR(32) PRIMARY KEY NOT NULL,
	password VARCHAR(512) NOT NULL
);
CREATE TABLE IF NOT EXISTS privileges (
	name VARCHAR(32) PRIMARY KEY NOT NULL,
	privileges VARCHAR(1024)
);
CREATE TABLE IF NOT EXISTS ban (
	addr VARCHAR(39) PRIMARY KEY NOT NULL,
	name VARCHAR(32) NOT NULL
);`, port)
	}

	db, ok := ConfKey("psql_db").(string)
	if !ok {
		return sqlite3()
	}

	host, ok := ConfKey("psql_host").(string)
	if !ok {
		log.Print("PostgreSQL host not set or not a string")
		return sqlite3()
	}

	port, ok := ConfKey("psql_port").(int)
	if !ok {
		log.Print("PostgreSQL port not set or not an integer")
		return sqlite3()
	}

	user, ok := ConfKey("psql_user").(string)
	if !ok {
		log.Print("PostgreSQL user not set or not a string")
		return sqlite3()
	}

	password, ok := ConfKey("psql_password").(string)
	if !ok {
		log.Print("PostgreSQL password not set or not a string")
		return sqlite3()
	}

	return psql(host, db, user, password, port)
}

// CreateUser creates a new entry in the authentication database
func CreateUser(name string, verifier, salt []byte) error {
	db, err := authDB()
	if err != nil {
		return err
	}
	defer db.Close()

	pwd := encodeVerifierAndSalt(salt, verifier)

	_, err = db.Exec(`INSERT INTO auth (
	name,
	password
) VALUES (
	?,
	?
);`, name, pwd)
	return err
}

// Password returns the SRP tokens of a user
func Password(name string) ([]byte, []byte, error) {
	db, err := authDB()
	if err != nil {
		return nil, nil, err
	}
	defer db.Close()

	var pwd string
	err = db.QueryRow(`SELECT password FROM auth WHERE name = ?;`, name).Scan(&pwd)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, nil, err
	}

	salt, verifier, err := decodeVerifierAndSalt(pwd)
	return verifier, salt, err
}

// SetPassword changes the SRP tokens of a user
func SetPassword(name string, verifier, salt []byte) error {
	db, err := authDB()
	if err != nil {
		return err
	}
	defer db.Close()

	pwd := encodeVerifierAndSalt(salt, verifier)

	_, err = db.Exec(`UPDATE auth SET password = ? WHERE name = ?;`, pwd, name)
	return err
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
		// This passphrase should not be changed wihtout deleting
		// the auth databases on the minetest servers
		err = SetStorageKey("auth:passphrase", string(passPhrase))
		if err != nil {
			log.Fatal(err)
		}
	} else {
		passPhrase = []byte(pwd)
	}
}
