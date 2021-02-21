package main

import (
	"errors"
	"net"
	"sync"

	"github.com/anon55555/mt/rudp"
)

var ErrPlayerLimitReached = errors.New("player limit reached")

type Listener struct {
	*rudp.Listener
	mu    sync.RWMutex
	peers map[*Peer]struct{}
}

var listener *Listener

func Listen(conn net.PacketConn) *Listener {
	return &Listener{
		Listener: rudp.Listen(conn),
		peers:    make(map[*Peer]struct{}),
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

	l.mu.Lock()
	l.peers[clt] = struct{}{}
	l.mu.Unlock()
	go func() {
		<-clt.Disco()

		l.mu.Lock()
		delete(l.peers, clt)
		l.mu.Unlock()
	}()

	clt.aoIDs = make(map[uint16]bool)
	clt.modChs = make(map[string]bool)
	clt.huds = make(map[uint32]bool)
	clt.sounds = make(map[int32]bool)
	clt.invlists = make(map[string]bool)

	maxPeers, ok := GetConfKey("player_limit").(int)
	if !ok {
		maxPeers = int(^uint(0) >> 1)
	}

	if GetPeerCount() >= maxPeers {
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
	connectedPeers++

	return clt, nil
}

// GetPeerByUsername returns the Peer that is using name for
// authentication
func (l *Listener) GetPeerByUsername(name string) *Peer {
	l.mu.RLock()
	defer l.mu.RUnlock()

	for peer := range l.peers {
		if peer.Username() == name {
			return peer
		}
	}

	return nil
}

// GetPeers returns an array containing all connected client Peers
func (l *Listener) GetPeers() []*Peer {
	l.mu.RLock()
	defer l.mu.RUnlock()

	var r []*Peer
	for p := range l.peers {
		r = append(r, p)
	}
	return r
}

// SetListener is used to make a listener available globally
// This can only be done once
func SetListener(l *Listener) {
	if listener == nil {
		listener = l
	}
}

// GetListener returns the global listener
func GetListener() *Listener {
	return listener
}
