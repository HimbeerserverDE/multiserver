package multiserver

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"sync"
)

var ErrPlayerLimitReached = errors.New("player limit reached")

type Listener struct {
	conn net.PacketConn

	clts chan cltPeer
	errs chan error

	mu        sync.Mutex
	addr2peer map[string]cltPeer
	id2peer   map[PeerID]cltPeer
	peerid    PeerID
}

var listener *Listener

// Listen listens for packets on conn until it is closed
func Listen(conn net.PacketConn) *Listener {
	l := &Listener{
		conn: conn,

		clts: make(chan cltPeer),
		errs: make(chan error),

		addr2peer: make(map[string]cltPeer),
		id2peer:   make(map[PeerID]cltPeer),
	}

	pkts := make(chan netPkt)
	go readNetPkts(l.conn, pkts, l.errs)
	go func() {
		for pkt := range pkts {
			if err := l.processNetPkt(pkt); err != nil {
				l.errs <- err
			}
		}

		close(l.clts)

		for _, clt := range l.addr2peer {
			clt.Close()
		}
	}()

	return l
}

// Accept waits for and returns a connecting Peer
// You should keep calling this until it returns ErrClosed
// so it doesn't leak a goroutine
func (l *Listener) Accept() (*Peer, error) {
	select {
	case clt, ok := <-l.clts:
		if !ok {
			select {
			case err := <-l.errs:
				return nil, err
			default:
				return nil, ErrClosed
			}
		}
		close(clt.accepted)

		connectedPeers++

		return clt.Peer, nil
	case err := <-l.errs:
		return nil, err
	}
}

// Addr returns the net.PacketConn the Listener is listening on
func (l *Listener) Conn() net.PacketConn { return l.conn }

var ErrOutOfPeerIDs = errors.New("out of peer ids")

type cltPeer struct {
	*Peer
	pkts     chan<- netPkt
	accepted chan struct{} // close-only
}

func (l *Listener) processNetPkt(pkt netPkt) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	addrstr := pkt.SrcAddr.String()

	clt, ok := l.addr2peer[addrstr]
	if !ok {
		prev := l.peerid
		for l.id2peer[l.peerid].Peer != nil || l.peerid < PeerIDCltMin {
			if l.peerid == prev-1 {
				return ErrOutOfPeerIDs
			}
			l.peerid++
		}

		pkts := make(chan netPkt, 256)

		clt = cltPeer{
			Peer:     newPeer(l.conn, pkt.SrcAddr, l.peerid, PeerIDSrv),
			pkts:     pkts,
			accepted: make(chan struct{}),
		}

		l.addr2peer[addrstr] = clt
		l.id2peer[clt.ID()] = clt

		data := make([]byte, 2+2)
		data[0] = uint8(rawTypeCtl)
		data[1] = uint8(ctlSetPeerID)
		binary.BigEndian.PutUint16(data[2:4], uint16(clt.ID()))
		if _, err := clt.sendRaw(rawPkt{Data: data}); err != nil {
			return fmt.Errorf("can't set client peer id: %w", err)
		}

		var maxPeers int
		maxPeersKey := GetConfKey("player_limit")
		if maxPeersKey == nil || fmt.Sprintf("%T", maxPeersKey) != "int" {
			maxPeers = -1
		}
		maxPeers = maxPeersKey.(int)

		if GetPeerCount() >= maxPeers && maxPeers > -1 {
			data := []byte{
				uint8(0x00), uint8(0x0A),
				uint8(0x06), uint8(0x00), uint8(0x00), uint8(0x00), uint8(0x00),
			}

			_, err := clt.Send(Pkt{Data: data, ChNo: 0, Unrel: false})
			if err != nil {
				return err
			}

			clt.SendDisco(0, true)
			clt.Close()

			return ErrPlayerLimitReached
		}

		go func() {
			select {
			case l.clts <- clt:
			case <-clt.Disco():
			}

			clt.processNetPkts(pkts)
		}()

		go func() {
			<-clt.Disco()

			l.mu.Lock()
			close(pkts)
			delete(l.addr2peer, addrstr)
			delete(l.id2peer, clt.ID())
			l.mu.Unlock()
		}()
	}

	select {
	case <-clt.accepted:
		clt.pkts <- pkt
	default:
		select {
		case clt.pkts <- pkt:
		default:
			return fmt.Errorf("ignoring net pkt from %s because buf is full", addrstr)
		}
	}

	return nil
}

func (l *Listener) GetPeerByID(id PeerID) *Peer {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	return l.id2peer[id].Peer
}

func SetListener(l *Listener) {
	listener = l
}

func GetListener() *Listener {
	return listener
}
