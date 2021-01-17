package multiserver

import "log"

// Proxy processes and forwards packets from src to dst
func Proxy(src, dst *Peer) {
	for {
		pkt, err := src.Recv()
		if !src.Forward() {
			return
		} else if !dst.Forward() {
			break
		}
		if err != nil {
			if err == ErrClosed {
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
