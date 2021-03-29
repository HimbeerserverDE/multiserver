package main

import (
	"bytes"
	"log"
	"os"
	"time"

	"github.com/anon55555/mt/rudp"
)

// End disconnects (from) all Peers and stops the process
func End(crash, reconnect bool) {
	log.Print("Ending")

	data := make([]byte, 7)
	data[0] = uint8(0x00)
	data[1] = uint8(ToClientAccessDenied)
	if crash {
		data[2] = uint8(AccessDeniedCrash)
	} else {
		data[2] = uint8(AccessDeniedShutdown)
	}
	data[3] = uint8(0x00)
	data[4] = uint8(0x00)
	if reconnect {
		data[5] = uint8(0x01)
	} else {
		data[5] = uint8(0x00)
	}
	data[6] = uint8(0x00)

	r := bytes.NewReader(data)

	for _, clt := range Conns() {
		_, err := clt.Send(rudp.Pkt{Reader: r})
		if err != nil {
			log.Print(err)
		}

		clt.Close()
	}

	rpcSrvMu.Lock()
	for srv := range rpcSrvs {
		srv.Close()
	}
	rpcSrvMu.Unlock()

	time.Sleep(time.Second)

	Announce(AnnounceDelete)

	if crash {
		os.Exit(1)
	} else {
		os.Exit(0)
	}
}
