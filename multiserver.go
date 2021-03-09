/*
Multiserver is a multi-server minetest reverse proxy capable of
media and definition multiplexing
*/
package main

import (
	"log"
	"net"
	"time"

	"github.com/anon55555/mt/rudp"
)

func main() {
	defaultSrv, ok := ConfKey("default_server").(string)
	if !ok {
		log.Fatal("Default server name not set or not a string")
	}

	defaultSrvAddr, ok := ConfKey("servers:" + defaultSrv + ":address").(string)
	if !ok {
		log.Fatal("Default server address not set or not a string")
	}

	host, ok := ConfKey("host").(string)
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

	l := Listen(lc)

	Announce(AnnounceStart)

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

		srv, err := Connect(conn, conn.RemoteAddr())
		if err != nil {
			data := []byte{
				0, ToClientAccessDenied,
				AccessDeniedServerFail, 0, 0, 0, 0,
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

		fin := make(chan *Peer)
		go Init(srv, clt, true, false, fin)

		go func() {
			srv = <-fin

			if srv == nil {
				data := []byte{
					0, ToClientAccessDenied,
					AccessDeniedServerFail, 0, 0, 0, 0,
				}

				select {
				case <-clt.Disco():
				default:
					ack, err := clt.Send(rudp.Pkt{Data: data})
					if err != nil {
						log.Print(err)
					}
					<-ack
				}

				clt.SendDisco(0, true)
				clt.Close()
				return
			}

			clt.SetServer(srv)

			go Proxy(clt, srv)
			go Proxy(srv, clt)
		}()
	}
}
