package main

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/anon55555/mt"
	"github.com/anon55555/mt/rudp"
)

var connectedPeers int = 0
var connectedPeersMu sync.RWMutex

// A Peer is a connection to a client or server
type Peer struct {
	*rudp.Peer

	username string
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

	initAoReceived   bool
	aoIDs            map[uint16]bool
	localPlayerCao   uint16
	currentPlayerCao uint16

	useRpcMu sync.RWMutex
	useRpc   bool
	noClt    bool
	modChs   map[string]bool

	huds map[uint32]bool

	sounds map[int32]bool

	inv *mt.Inv
}

// Username returns the username of the Peer
// if it isn't a server
func (p *Peer) Username() string { return p.username }

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

// UseRpc reports whether RPC messages can be sent to the Peer
func (p *Peer) UseRpc() bool {
	p.useRpcMu.RLock()
	defer p.useRpcMu.RUnlock()

	return p.useRpc
}

// SetUseRpc sets the value returned by UseRpc
func (p *Peer) SetUseRpc(useRpc bool) {
	p.useRpcMu.Lock()
	defer p.useRpcMu.Unlock()

	p.useRpc = useRpc
}

// NoClt reports whether the Peer is RPC-only
func (p *Peer) NoClt() bool { return p.noClt }

// MakeRpcOnly marks the Peer as RPC-only
func (p *Peer) MakeRpcOnly() {
	p.noClt = true
}

// Inv returns the inventory of the Peer
func (p *Peer) Inv() *mt.Inv { return p.inv }

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

		return nil, fmt.Errorf("server at %s is unreachable", addr.String())
	case <-ack:
	}

	return srv, nil
}

// PeerCount reports how many client Peers are connected
func PeerCount() int {
	connectedPeersMu.RLock()
	defer connectedPeersMu.RUnlock()

	return connectedPeers
}
