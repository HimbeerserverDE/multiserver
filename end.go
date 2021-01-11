package multiserver

import (
	"log"
	"os"
	"time"
)

func End(crash, reconnect bool) {
	log.Print("Ending")

	l := GetListener()

	data := make([]byte, 7)
	data[0] = uint8(0x00)
	data[1] = uint8(0x0A)
	if crash {
		data[2] = uint8(0x0C)
	} else {
		data[2] = uint8(0x0B)
	}
	data[3] = uint8(0x00)
	data[4] = uint8(0x00)
	if reconnect {
		data[5] = uint8(0x01)
	} else {
		data[5] = uint8(0x00)
	}
	data[6] = uint8(0x00)

	i := PeerIDCltMin
	l.mu.Lock()
	for l.id2peer[i].Peer != nil {
		ack, err := l.id2peer[i].Send(Pkt{Data: data})
		if err != nil {
			log.Print(err)
		}
		<-ack

		l.id2peer[i].SendDisco(0, true)
		l.id2peer[i].Close()

		i++
	}
	l.mu.Unlock()

	time.Sleep(time.Second)

	if crash {
		os.Exit(1)
	} else {
		os.Exit(0)
	}
}
