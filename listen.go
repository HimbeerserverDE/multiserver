package main

import (
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

var peerMu sync.RWMutex
var peers map[*Peer]struct{}

func Listen(conn net.PacketConn) *Listener {
	return &Listener{
		Listener: rudp.Listen(conn),
	}
}

// Accept waits for and returns a connecting Peer
// You should keep calling this until it returns ErrClosed
// so it doesn't leak a goroutine
func (l *Listener) Accept() (*Peer, error) {
	rp, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}

	clt := &Peer{Peer: rp}

	peerMu.Lock()
	peers[clt] = struct{}{}
	peerMu.Unlock()

	go func() {
		<-clt.Disco()

		peerMu.Lock()
		delete(peers, clt)
		peerMu.Unlock()
	}()

	clt.aoIDs = make(map[uint16]bool)
	clt.modChs = make(map[string]bool)
	clt.huds = make(map[uint32]bool)
	clt.sounds = make(map[int32]bool)
	clt.inv = &mt.Inv{}

	maxPeers, ok := ConfKey("player_limit").(int)
	if !ok {
		maxPeers = int(^uint(0) >> 1)
	}

	if PeerCount() >= maxPeers {
		data := []byte{
			0, ToClientAccessDenied,
			AccessDeniedTooManyUsers, 0, 0, 0, 0,
		}

		_, err := clt.Send(rudp.Pkt{Data: data})
		if err != nil {
			return nil, err
		}

		return nil, ErrPlayerLimitReached
	}

	connectedPeersMu.Lock()
	connectedPeers++
	connectedPeersMu.Unlock()

	return clt, nil
}

// PeerByUsername returns the Peer that is using the specified name
// for authentication
func PeerByUsername(name string) *Peer {
	peerMu.RLock()
	defer peerMu.RUnlock()

	for p := range peers {
		if p.Username() == name {
			return p
		}
	}

	return nil
}

// Peers returns an array containing all connected client Peers
func Peers() []*Peer {
	peerMu.RLock()
	defer peerMu.RUnlock()

	var r []*Peer
	for p := range peers {
		r = append(r, p)
	}
	return r
}

func init() {
	peerMu.Lock()
	defer peerMu.Unlock()

	peers = make(map[*Peer]struct{})
}
