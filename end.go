package main

import (
	"log"
	"os"
	"time"

	"github.com/tncardoso/gocurses"
)

// End disconnects (from) all Peers and stops the process
func End(crash, reconnect bool) {
	log.Print("Ending")

	var reason uint8 = AccessDeniedShutdown
	if crash {
		reason = AccessDeniedCrash
	}

	for _, clt := range Conns() {
		clt.CloseWith(reason, "", reconnect)
	}

	rpcSrvMu.Lock()
	for srv := range rpcSrvs {
		srv.Close()
	}
	rpcSrvMu.Unlock()

	time.Sleep(time.Second)

	Announce(AnnounceDelete)

	gocurses.End()

	if crash {
		os.Exit(1)
	}
	os.Exit(0)
}
