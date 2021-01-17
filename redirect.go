package multiserver

import (
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"net"
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

	straddr := GetConfKey("servers:" + newsrv + ":address")
	if straddr == nil || fmt.Sprintf("%T", straddr) != "string" {
		return ErrServerDoesNotExist
	}

	if p.Server().Addr().String() == straddr {
		return ErrAlreadyConnected
	}

	srvaddr, err := net.ResolveUDPAddr("udp", straddr.(string))
	if err != nil {
		return err
	}

	conn, err := net.DialUDP("udp", nil, srvaddr)
	if err != nil {
		return err
	}
	srv := Connect(conn, conn.RemoteAddr())

	if srv == nil {
		return ErrServerUnreachable
	}

	// Remove active objects
	aolen := 0
	for range aoIDs[p.ID()] {
		aolen++
	}

	data := make([]byte, 6+aolen*2)
	data[0] = uint8(0x00)
	data[1] = uint8(ToClientActiveObjectRemoveAdd)
	binary.BigEndian.PutUint16(data[2:4], uint16(aolen))
	i := 4
	for ao := range aoIDs[p.ID()] {
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

			ack, err := p.Send(Pkt{Data: data})
			if err != nil {
				return err
			}
			<-ack
		}
	}

	ack, err := p.Send(Pkt{Data: data})
	if err != nil {
		return err
	}
	<-ack

	aoIDs[p.ID()] = make(map[uint16]bool)

	p.Server().stopForwarding()

	fin := make(chan struct{}) // close-only
	go Init(p, srv, true, false, fin)
	<-fin

	p.SetServer(srv)

	go Proxy(p, srv)
	go Proxy(srv, p)

	log.Print(p.Addr().String() + " redirected to " + newsrv)

	return nil
}
