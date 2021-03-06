package main

import (
	"database/sql"
	"encoding/binary"
	"errors"
	"fmt"
	"net"

	"github.com/anon55555/mt/rudp"
	_ "github.com/mattn/go-sqlite3"
)

var ErrInvalidAddress = errors.New("invalid ip address format")

// addBanItem inserts a ban DB entry
func addBanItem(db *sql.DB, addr, name string) error {
	sql_addBanItem := `INSERT INTO ban (
		addr,
		name
	) VALUES (
		?,
		?
	);
	`

	stmt, err := db.Prepare(sql_addBanItem)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(addr, name)
	if err != nil {
		return err
	}

	return nil
}

// readBanItem selects and reads a ban DB entry
func readBanItem(db *sql.DB, addr string) (string, error) {
	sql_readBanItem := `SELECT name FROM ban WHERE addr = ?;`

	stmt, err := db.Prepare(sql_readBanItem)
	if err != nil {
		return "", err
	}
	defer stmt.Close()

	rows, err := stmt.Query(addr)
	if err != nil {
		return "", err
	}

	var r string

	for rows.Next() {
		err = rows.Scan(&r)
	}

	return r, err
}

// deleteBanItem deletes a ban DB entry
func deleteBanItem(db *sql.DB, name string) error {
	sql_deleteBanItem := `DELETE FROM ban WHERE name = ? OR addr = ?;`

	stmt, err := db.Prepare(sql_deleteBanItem)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(name, name)
	if err != nil {
		return err
	}

	return nil
}

// BanList returns the list of banned players and IP addresses
func BanList() (map[string]string, error) {
	sql_readBanItems := `SELECT addr, name FROM ban;`

	db, err := initAuthDB()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	stmt, err := db.Prepare(sql_readBanItems)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.Query()
	if err != nil {
		return nil, err
	}

	r := make(map[string]string)

	for rows.Next() {
		var addr, name string

		if err = rows.Scan(&addr, &name); err != nil {
			return nil, err
		}

		r[addr] = name
	}

	return r, nil
}

// IsBanned reports whether a Peer is banned
func (p *Peer) IsBanned() (bool, string, error) {
	db, err := initAuthDB()
	if err != nil {
		return true, "", err
	}
	defer db.Close()

	addr := p.Addr().(*net.UDPAddr).IP.String()

	name, err := readBanItem(db, addr)
	if err != nil {
		return true, "", err
	}

	return name != "", name, nil
}

// Ban adds a Peer to the ban list
func (p *Peer) Ban() error {
	banned, _, err := p.IsBanned()
	if err != nil {
		return err
	}

	if banned {
		return fmt.Errorf("ip address %s is already banned", p.Addr().String())
	}

	db, err := initAuthDB()
	if err != nil {
		return err
	}
	defer db.Close()

	name := p.Username()
	addr := p.Addr().(*net.UDPAddr).IP.String()

	err = addBanItem(db, addr, name)
	if err != nil {
		return err
	}

	reason := []byte("Banned.")
	l := len(reason)

	data := make([]byte, 7+l)
	data[0] = uint8(0x00)
	data[1] = uint8(ToClientAccessDenied)
	data[2] = uint8(AccessDeniedCustomString)
	binary.BigEndian.PutUint16(data[3:5], uint16(l))
	copy(data[5:5+l], reason)
	data[5+l] = uint8(0x00)
	data[6+l] = uint8(0x00)

	ack, err := p.Send(rudp.Pkt{Data: data})
	if err != nil {
		return err
	}
	<-ack

	p.SendDisco(0, true)
	p.Close()

	return nil
}

func Ban(addr string) error {
	db, err := initAuthDB()
	if err != nil {
		return err
	}
	defer db.Close()

	name := "not known"

	if net.ParseIP(addr) == nil {
		return ErrInvalidAddress
	}

	err = addBanItem(db, addr, name)
	if err != nil {
		return err
	}

	return nil
}

func Unban(name string) error {
	db, err := initAuthDB()
	if err != nil {
		return err
	}
	defer db.Close()

	err = deleteBanItem(db, name)
	if err != nil {
		return err
	}

	return nil
}
