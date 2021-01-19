/*
Multiserver is a multi-server minetest reverse proxy capable of
media multiplexing
*/
package main

import (
	"log"
	"net"
	"time"

	"github.com/anon55555/mt/rudp"

	"github.com/HimbeerserverDE/multiserver"
)

func main() {
	defaultSrv, ok := multiserver.GetConfKey("default_server").(string)
	if !ok {
		log.Fatal("Default server name not set or not a string")
	}

	defaultSrvAddr, ok := multiserver.GetConfKey("servers:" + defaultSrv + ":address").(string)
	if !ok {
		log.Fatal("Default server address not set or not a string")
	}

	host, ok := multiserver.GetConfKey("host").(string)
	if !ok {
		log.Fatal("Host not set or not a string")
	}

	srvaddr, err := net.ResolveUDPAddr("udp", defaultSrvAddr)
	if err != nil {
		log.Fatal(err)
	}

	lc, err := net.ListenPacket("udp", host)
	if err != nil {
		log.Fatal(err)
	}
	defer lc.Close()

	log.Print("Listening on " + host)

	l := multiserver.Listen(lc)
	multiserver.SetListener(l)
	for {
		clt, err := l.Accept()
		if err != nil {
			log.Print(err)
			continue
		}

		log.Print(clt.Addr(), " connected")

		conn, err := net.DialUDP("udp", nil, srvaddr)
		if err != nil {
			log.Fatal(err)
		}

		srv, err := multiserver.Connect(conn, conn.RemoteAddr())
		if err != nil {
			data := []byte{
				0, multiserver.ToClientAccessDenied,
				multiserver.AccessDeniedServerFail, 0, 0, 0, 0,
			}

			_, err := clt.Send(rudp.Pkt{Data: data})
			if err != nil {
				log.Print(err)
			}

			time.Sleep(250 * time.Millisecond)

			clt.SendDisco(0, true)
			clt.Close()

			continue
		}

		fin := make(chan struct{}) // close-only
		go multiserver.Init(srv, clt, true, false, fin)

		go func() {
			<-fin

			clt.SetServer(srv)

			go multiserver.Proxy(clt, srv)
			go multiserver.Proxy(srv, clt)
		}()
	}
}
