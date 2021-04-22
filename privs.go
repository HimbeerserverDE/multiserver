package main

import (
	"database/sql"
	"errors"
	"log"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

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

// Privs returns the privileges of a player
func Privs(name string) (map[string]bool, error) {
	db, err := authDB()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	var eprivs string
	err = db.QueryRow(`SELECT privileges FROM privileges WHERE name = $1;`, name).Scan(&eprivs)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return make(map[string]bool), err
	}

	return decodePrivs(eprivs), nil
}

// Privs returns the privileges of a Conn
func (c *Conn) Privs() (map[string]bool, error) {
	return Privs(c.Username())
}

// SetPrivs sets the privileges of a player
func SetPrivs(name string, privs map[string]bool) error {
	db, err := authDB()
	if err != nil {
		return err
	}
	defer db.Close()

	_, err = db.Exec(`INSERT INTO privileges (
	name,
	privileges
) VALUES (
	$1,
	$2
);`, name, encodePrivs(privs))
	_, err = db.Exec(`UPDATE privileges SET privileges = $1 WHERE name = $2;`, encodePrivs(privs), name)

	return err
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
		privs, err := Privs(admin)
		if err != nil {
			log.Print(err)
			return
		}

		privs["privs"] = true

		if err = SetPrivs(admin, privs); err != nil {
			log.Print(err)
			return
		}
	}
}
