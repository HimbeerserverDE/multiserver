package multiserver

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
	sql_addPrivItem := `INSERT INTO privileges (
		name,
		privileges
	) VALUES (
		?,
		""
	);
	`

	stmt, err := db.Prepare(sql_addPrivItem)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(name)
	if err != nil {
		return err
	}

	return nil
}

// modPrivItem updates a priv DB entry
func modPrivItem(db *sql.DB, name, privs string) error {
	sql_modPrivItem := `UPDATE privileges SET privileges = ? WHERE name = ?;`

	stmt, err := db.Prepare(sql_modPrivItem)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(privs, name)
	if err != nil {
		return err
	}

	return nil
}

// readPrivItem selects and reads a priv DB entry
func readPrivItem(db *sql.DB, name string) (string, error) {
	sql_readPrivItem := `SELECT privileges FROM privileges WHERE name = ?;`

	stmt, err := db.Prepare(sql_readPrivItem)
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

// GetPrivs returns the privileges of the Peer
func (p *Peer) GetPrivs() (map[string]bool, error) {
	db, err := initAuthDB()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	eprivs, err := readPrivItem(db, string(p.username))
	if err != nil {
		return nil, err
	}

	return decodePrivs(eprivs), nil
}

// SetPrivs sets the privileges for the Peer
func (p *Peer) SetPrivs(privs map[string]bool) error {
	db, err := initAuthDB()
	if err != nil {
		return err
	}
	defer db.Close()

	err = modPrivItem(db, string(p.username), encodePrivs(privs))
	if err != nil {
		return err
	}

	return nil
}

// CheckPrivs reports if the Peer has all ofthe specified privileges
func (p *Peer) CheckPrivs(req map[string]bool) (bool, error) {
	privs, err := p.GetPrivs()
	if err != nil {
		return false, err
	}

	allow := true
	for priv := range req {
		if req[priv] && !privs[priv] {
			allow = false
			break
		}
	}

	return allow, nil
}

func init() {
	if admin, ok := GetConfKey("admin").(string); ok {
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
