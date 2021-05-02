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

	time.Sleep(time.Second)

	rpcSrvMu.Lock()
	for srv := range rpcSrvs {
		srv.Close()
	}
	rpcSrvMu.Unlock()

	Announce(AnnounceDelete)

	log.Writer().(*Logger).Close()
	gocurses.End()

	if crash {
		os.Exit(1)
	}
	os.Exit(0)
}
