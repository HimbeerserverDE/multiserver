package main

import (
	"fmt"
	"log"
	"net"
	"time"

	"github.com/HimbeerserverDE/multiserver"
)

func main() {
	defaultSrv := multiserver.GetConfKey("default_server")
	if defaultSrv == nil || fmt.Sprintf("%T", defaultSrv) != "string" {
		log.Fatal("Default server name not set or not a string")
		return
	}

	defaultSrvAddr := multiserver.GetConfKey("servers:" + defaultSrv.(string) + ":address")
	if defaultSrvAddr == nil || fmt.Sprintf("%T", defaultSrvAddr) != "string" {
		log.Fatal("Default server address not set or not a string")
		return
	}

	host := multiserver.GetConfKey("host")
	if host == nil || fmt.Sprintf("%T", host) != "string" {
		log.Fatal("Host not set or not a string")
		return
	}

	srvaddr, err := net.ResolveUDPAddr("udp", defaultSrvAddr.(string))
	if err != nil {
		log.Fatal(err)
		return
	}

	lc, err := net.ListenPacket("udp", host.(string))
	if err != nil {
		log.Fatal(err)
		return
	}
	defer lc.Close()

	log.Print("Listening on " + host.(string))

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
			return
		}
		srv := multiserver.Connect(conn, conn.RemoteAddr())

		if srv == nil {
			data := []byte{
				uint8(0x00), uint8(multiserver.ToClientAccessDenied),
				uint8(multiserver.AccessDeniedServerFail), uint8(0x00), uint8(0x00), uint8(0x00), uint8(0x00),
			}

			_, err := clt.Send(multiserver.Pkt{Data: data})
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
