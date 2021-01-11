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

// Redirect closes the connection to srv1
// and redirects the client to srv2
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
	len := 0
	for _ = range aoIDs[p.ID()] {
		len++
	}

	data := make([]byte, 6+len*2)
	data[0] = uint8(0x00)
	data[1] = uint8(0x31)
	binary.BigEndian.PutUint16(data[2:4], uint16(len))
	i := 4
	for ao := range aoIDs[p.ID()] {
		binary.BigEndian.PutUint16(data[i:2+i], ao)

		i += 2
	}
	binary.BigEndian.PutUint16(data[i:2+i], uint16(0))

	ack, err := p.Send(Pkt{Data: data})
	if err != nil {
		return err
	}
	<-ack

	aoIDs[p.ID()] = make(map[uint16]bool)

	p.Server().StopForwarding()

	fin := make(chan struct{}) // close-only
	go Init(p, srv, true, fin)
	<-fin

	p.SetServer(srv)

	go Proxy(p, srv)
	go Proxy(srv, p)

	log.Print(p.Addr().String() + " redirected to " + newsrv)

	return nil
}
