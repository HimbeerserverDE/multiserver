package main

import (
	"errors"
	"net"
	"sync"
	"time"

	"github.com/anon55555/mt/rudp"
)

var ErrServerUnreachable = errors.New("server is unreachable")

var connectedPeers int = 0
var connectedPeersMu sync.RWMutex

// A Peer is a connection to a client or server
type Peer struct {
	*rudp.Peer

	username []byte
	srp_s    []byte
	srp_A    []byte
	srp_a    []byte
	srp_B    []byte
	srp_K    []byte
	authMech int
	sudoMode bool

	stopforward bool
	forwardMu   sync.RWMutex

	redirectMu sync.Mutex
	srvMu      sync.RWMutex
	srv        *Peer

	initAoReceived bool
	aoIDs          map[uint16]bool

	useRpc bool
	modChs map[string]bool
}

// Username returns the username of the Peer
// if it isn't a server
func (p *Peer) Username() string { return string(p.username) }

// Forward reports whether the Proxy func should continue or stop
func (p *Peer) Forward() bool {
	p.forwardMu.RLock()
	defer p.forwardMu.RUnlock()

	return !p.stopforward
}

// stopForwarding tells the Proxy func to stop
func (p *Peer) stopForwarding() {
	p.forwardMu.Lock()
	defer p.forwardMu.Unlock()

	p.stopforward = true
}

// Server returns the Peer this Peer is connected to
// if it isn't a server
func (p *Peer) Server() *Peer {
	p.srvMu.RLock()
	defer p.srvMu.RUnlock()

	return p.srv
}

// ServerName returns the name of the Peer this Peer is connected to
// if this Peer is not a server
func (p *Peer) ServerName() string {
	servers := GetConfKey("servers").(map[interface{}]interface{})
	for server := range servers {
		if GetConfKey("servers:"+server.(string)+":address") == p.Server().Addr().String() {
			return server.(string)
		}
	}

	return ""
}

// SetServer sets the Peer this Peer is connected to
// if this Peer is not a server
func (p *Peer) SetServer(s *Peer) {
	p.srvMu.Lock()
	defer p.srvMu.Unlock()

	p.srv = s
}

// Connect connects to the server on conn
// and closes conn when the Peer disconnects
func Connect(conn net.PacketConn, addr net.Addr) (*Peer, error) {
	srv := &Peer{Peer: rudp.Connect(conn, addr)}

	ack, err := srv.Send(rudp.Pkt{Data: []byte{0, 0}})
	if err != nil {
		return nil, err
	}

	select {
	case <-time.After(8 * time.Second):
		srv.SendDisco(0, true)
		srv.Close()

		return nil, ErrServerUnreachable
	case <-ack:
	}

	return srv, nil
}

// GetPeerCount reports how many client Peers are connected
func GetPeerCount() int {
	connectedPeersMu.RLock()
	defer connectedPeersMu.RUnlock()

	return connectedPeers
}
