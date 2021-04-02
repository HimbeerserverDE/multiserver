package main

import (
	"errors"
	"log"
	"net"
)

// Proxy processes and forwards packets from src to dst
func Proxy(src, dst *Conn) {
	if src == nil {
		dst.CloseWith(AccessDeniedServerFail, "", false)
		processLeave(dst)
		return
	} else if dst == nil {
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
				if err = src.WhyClosed(); err != nil {
					log.Print(src.Addr().String(), " disconnected with error: ", err)
				} else {
					log.Print(src.Addr().String(), " disconnected")
				}

				if !src.IsSrv() {
					connectedConnsMu.Lock()
					connectedConns--
					connectedConnsMu.Unlock()

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

	dst.Close()
}
