package main

import (
	"database/sql"
	"log"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

// encodePrivs encodes priv map into DB-ready string
func encodePrivs(privs map[string]bool) string {
	lenP := 0
	for priv := range privs {
		if privs[priv] {
			lenP++
		}
	}

	ps := make([]string, lenP)

	i := 0
	for priv := range privs {
		if privs[priv] {
			ps[i] = priv

			i++
		}
	}

	r := strings.Join(ps, "|")

	return r
}

// decodePrivs decodes DB-ready string into priv map
func decodePrivs(s string) map[string]bool {
	ps := strings.Split(s, "|")

	r := make(map[string]bool)

	for i := range ps {
		if ps[i] != "" {
			r[ps[i]] = true
		}
	}

	return r
}

// addPrivItem inserts a priv DB entry
func addPrivItem(db *sql.DB, name string) error {
	_, err := db.Exec(`INSERT INTO privileges (
		name,
		privileges
	) VALUES (
		?,
		""
	);`, name)
	return nil
}

// modPrivItem updates a priv DB entry
func modPrivItem(db *sql.DB, name, privs string) error {
	_, err := db.Exec(`UPDATE privileges SET privileges = ? WHERE name = ?;`, name)
	return nil
}

// readPrivItem selects and reads a priv DB entry
func readPrivItem(db *sql.DB, name string) (string, error) {
	var r string
	err := db.QueryRow(`SELECT privileges FROM privileges WHERE name = ?;`, name).Scan(&r)
	return r, err
}

// Privs returns the privileges of a player
func Privs(name string) (map[string]bool, error) {
	db, err := initAuthDB()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	eprivs, err := readPrivItem(db, name)
	if err != nil {
		return nil, err
	}

	return decodePrivs(eprivs), nil
}

// Privs returns the privileges of a Conn
func (c *Conn) Privs() (map[string]bool, error) {
	return Privs(c.Username())
}

// SetPrivs sets the privileges of a player
func SetPrivs(name string, privs map[string]bool) error {
	db, err := initAuthDB()
	if err != nil {
		return err
	}
	defer db.Close()

	err = modPrivItem(db, name, encodePrivs(privs))
	if err != nil {
		return err
	}

	return nil
}

// SetPrivs sets the privileges of a Conn
func (c *Conn) SetPrivs(privs map[string]bool) error {
	return SetPrivs(c.Username(), privs)
}

// CheckPrivs reports if a player has all of the specified privileges
func CheckPrivs(name string, req map[string]bool) (bool, error) {
	privs, err := Privs(name)
	if err != nil {
		return false, err
	}

	for priv := range req {
		if req[priv] && !privs[priv] {
			return false, nil
		}
	}

	return true, nil
}

// CheckPrivs reports if a Conn has all of the specified privileges
func (c *Conn) CheckPrivs(req map[string]bool) (bool, error) {
	return CheckPrivs(c.Username(), req)
}

func init() {
	if admin, ok := ConfKey("admin").(string); ok {
		db, err := initAuthDB()
		if err != nil {
			log.Print(err)
			return
		}

		eprivs, err := readPrivItem(db, admin)
		if err != nil {
			log.Print(err)
			return
		}

		privs := decodePrivs(eprivs)
		privs["privs"] = true

		newprivs := encodePrivs(privs)

		modPrivItem(db, admin, newprivs)
	}
}
