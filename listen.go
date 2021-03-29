package main

import (
	"bytes"
	"errors"
	"net"
	"sync"

	"github.com/anon55555/mt"
	"github.com/anon55555/mt/rudp"
)

var ErrPlayerLimitReached = errors.New("player limit reached")

type Listener struct {
	*rudp.Listener
}

var connMu sync.RWMutex
var conns map[*Conn]struct{}

func Listen(conn net.PacketConn) *Listener {
	return &Listener{
		Listener: rudp.Listen(conn),
	}
}

// Accept waits for and returns a connecting Conn
// You should keep calling this until it returns ErrClosed
// so it doesn't leak a goroutine
func (l *Listener) Accept() (*Conn, error) {
	rp, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}

	clt := &Conn{Conn: rp}

	connMu.Lock()
	conns[clt] = struct{}{}
	connMu.Unlock()

	go func() {
		<-clt.Closed()

		connMu.Lock()
		delete(conns, clt)
		connMu.Unlock()
	}()

	clt.aoIDs = make(map[uint16]bool)
	clt.modChs = make(map[string]bool)
	clt.huds = make(map[uint32]bool)
	clt.sounds = make(map[int32]bool)
	clt.inv = &mt.Inv{}

	maxConns, ok := ConfKey("player_limit").(int)
	if !ok {
		maxConns = int(^uint(0) >> 1)
	}

	if ConnCount() >= maxConns {
		data := []byte{
			0, ToClientAccessDenied,
			AccessDeniedTooManyUsers, 0, 0, 0, 0,
		}

		_, err := clt.Send(rudp.Pkt{Reader: bytes.NewReader(data)})
		if err != nil {
			return nil, err
		}

		return nil, ErrPlayerLimitReached
	}

	connectedConnsMu.Lock()
	connectedConns++
	connectedConnsMu.Unlock()

	return clt, nil
}

// ConnByUsername returns the Conn that is using the specified name
// for authentication
func ConnByUsername(name string) *Conn {
	connMu.RLock()
	defer connMu.RUnlock()

	for c := range conns {
		if c.Username() == name {
			return c
		}
	}

	return nil
}

// Conns returns an array containing all connected client Conns
func Conns() []*Conn {
	connMu.RLock()
	defer connMu.RUnlock()

	var r []*Conn
	for c := range conns {
		r = append(r, c)
	}
	return r
}

func init() {
	connMu.Lock()
	defer connMu.Unlock()

	conns = make(map[*Conn]struct{})
}
