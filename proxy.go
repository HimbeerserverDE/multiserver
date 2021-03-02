package main

import (
	"errors"
	"log"
	"net"

	"github.com/anon55555/mt/rudp"
)

// Proxy processes and forwards packets from src to dst
func Proxy(src, dst *Peer) {
	if src == nil {
		data := []byte{
			0, ToClientAccessDenied,
			AccessDeniedServerFail, 0, 0, 0, 0,
		}

		_, err := dst.Send(rudp.Pkt{Data: data})
		if err != nil {
			log.Print(err)
		}

		dst.SendDisco(0, true)
		dst.Close()
		processLeave(dst)

		return
	} else if dst == nil {
		src.SendDisco(0, true)
		src.Close()

		return
	}

	for {
		pkt, err := src.Recv()
		if !src.Forward() {
			return
		} else if !dst.Forward() {
			break
		}
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				msg := src.Addr().String() + " disconnected"
				if src.TimedOut() {
					msg += " (timed out)"
				}
				log.Print(msg)

				if !src.IsSrv() {
					connectedPeersMu.Lock()
					connectedPeers--
					connectedPeersMu.Unlock()

					processLeave(src)
				}

				break
			}

			log.Print(err)
			continue
		}

		// Process
		if processPktCommand(src, dst, &pkt) {
			continue
		}

		// Forward
		if _, err := dst.Send(pkt); err != nil {
			log.Print(err)
		}
	}

	dst.SendDisco(0, true)
	dst.Close()
}
