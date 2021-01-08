package multiserver

import (
	"strings"
	"database/sql"
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
		r[ps[i]] = true
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