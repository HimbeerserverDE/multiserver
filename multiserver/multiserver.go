package main

import (
	"net"
	"log"
	"fmt"
	
	"github.com/HimbeerserverDE/multiserver"
)

func main() {
	multiserver.LoadConfig()
	
	multiserver.InitLua()
	defer multiserver.CloseLua()
	
	err := multiserver.LoadPlugins()
	if err != nil {
		log.Fatal(err)
		return
	}
	
	lobbyaddr := multiserver.GetConfKey("servers:lobby:address")
	if lobbyaddr == nil || fmt.Sprintf("%T", lobbyaddr) != "string" {
		log.Fatal("Lobby server address not set or not a string")
		return
	}
	
	host := multiserver.GetConfKey("host")
	if host == nil || fmt.Sprintf("%T", host) != "string" {
		log.Fatal("Host not set or not a string")
		return
	}
	
	srvaddr, err := net.ResolveUDPAddr("udp", lobbyaddr.(string))
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
		
		errs := make(chan error)
		fin := make(chan struct{}) // close-only
		go multiserver.Init(srv, clt, false, errs, fin)
		<-fin
		
		clt.SetServer(srv)
		
		go multiserver.Proxy(clt, srv)
		go multiserver.Proxy(srv, clt)
	}
}
