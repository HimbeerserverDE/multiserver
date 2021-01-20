package multiserver

import (
	"encoding/binary"
	"errors"
	"log"
	"net"

	"github.com/anon55555/mt/rudp"
)

var ErrServerDoesNotExist = errors.New("server doesn't exist")
var ErrAlreadyConnected = errors.New("already connected to server")

var onRedirectDone []func(*Peer, string, bool)

// RegisterOnRedirectDone registers a callback function that is called
// when the Peer.Redirect method exits
func RegisterOnRedirectDone(function func(*Peer, string, bool)) {
	onRedirectDone = append(onRedirectDone, function)
}

func processRedirectDone(p *Peer, newsrv string) {
	success := p.ServerName() == newsrv

	for i := range onRedirectDone {
		onRedirectDone[i](p, newsrv, success)
	}
}

// Redirect sends the Peer to the minetest server named newsrv
func (p *Peer) Redirect(newsrv string) error {
	p.redirectMu.Lock()
	defer p.redirectMu.Unlock()

	defer processRedirectDone(p, newsrv)

	straddr, ok := GetConfKey("servers:" + newsrv + ":address").(string)
	if !ok {
		return ErrServerDoesNotExist
	}

	if p.Server().Addr().String() == straddr {
		return ErrAlreadyConnected
	}

	srvaddr, err := net.ResolveUDPAddr("udp", straddr)
	if err != nil {
		return err
	}

	conn, err := net.DialUDP("udp", nil, srvaddr)
	if err != nil {
		return err
	}

	srv, err := Connect(conn, conn.RemoteAddr())
	if err != nil {
		return err
	}

	// Remove active objects
	data := make([]byte, 6+len(p.aoIDs)*2)
	data[0] = uint8(0x00)
	data[1] = uint8(ToClientActiveObjectRemoveAdd)
	binary.BigEndian.PutUint16(data[2:4], uint16(len(p.aoIDs)))
	i := 4
	for ao := range p.aoIDs {
		binary.BigEndian.PutUint16(data[i:2+i], ao)

		i += 2
	}
	binary.BigEndian.PutUint16(data[i:2+i], uint16(0))

	if len(detachedinvs[newsrv]) > 0 {
		for i := range detachedinvs[newsrv] {
			data := make([]byte, 2+len(detachedinvs[newsrv][i]))
			data[0] = uint8(0x00)
			data[1] = uint8(ToClientDetachedInventory)
			copy(data[2:], detachedinvs[newsrv][i])

			ack, err := p.Send(rudp.Pkt{Data: data})
			if err != nil {
				return err
			}
			<-ack
		}
	}

	ack, err := p.Send(rudp.Pkt{Data: data})
	if err != nil {
		return err
	}
	<-ack

	p.aoIDs = make(map[uint16]bool)

	p.Server().stopForwarding()

	fin := make(chan *Peer) // close-only
	go Init(p, srv, true, false, fin)
	<-fin

	p.SetServer(srv)

	go Proxy(p, srv)
	go Proxy(srv, p)

	log.Print(p.Addr().String() + " redirected to " + newsrv)

	return nil
}
