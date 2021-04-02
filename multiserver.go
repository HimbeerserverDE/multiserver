/*
Multiserver is a multi-server minetest reverse proxy capable of
media and definition multiplexing
*/
package main

import (
	"log"
	"net"
)

func main() {
	defaultSrv, ok := ConfKey("default_server").(string)
	if !ok {
		log.Fatal("Default server name not set or not a string")
	}

	_, ok = ConfKey("servers:" + defaultSrv + ":address").(string)
	if !ok {
		log.Fatal("Default server address not set or not a string")
	}

	host, ok := ConfKey("host").(string)
	if !ok {
		host = "0.0.0.0:33000"
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

		fin := make(chan *Conn)
		go Init(nil, clt, true, false, fin)

		go func() {
			srv := <-fin

			if srv == nil {
				select {
				case <-clt.Closed():
					clt.Close()
				default:
					clt.CloseWith(AccessDeniedServerFail, "", false)
				}

				return
			}

			clt.SetServer(srv)

			go Proxy(clt, srv)
			go Proxy(srv, clt)
		}()
	}
}
