package multiserver

import (
	"crypto/subtle"
	"encoding/binary"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/HimbeerserverDE/srp"
)

var ErrAuthFailed = errors.New("authentication failure")

// Init authenticates to the server srv
// and finishes the initialisation process if ignMedia is true
// This doesn't support AUTH_MECHANISM_FIRST_SRP yet
func Init(p, p2 *Peer, ignMedia bool, fin chan struct{}) {
	defer close(fin)

	if p2.ID() == PeerIDSrv {
		// We're trying to connect to a server
		// INIT
		data := make([]byte, 11+len(p.username))
		data[0] = uint8(0x00)
		data[1] = uint8(0x02)
		data[2] = uint8(0x1c)
		binary.BigEndian.PutUint16(data[3:5], uint16(0x0000))
		binary.BigEndian.PutUint16(data[5:7], uint16(0x0025))
		binary.BigEndian.PutUint16(data[7:9], uint16(0x0027))
		binary.BigEndian.PutUint16(data[9:11], uint16(len(p.username)))
		copy(data[11:], p.username)

		time.Sleep(250 * time.Millisecond)

		if _, err := p2.Send(Pkt{Data: data, ChNo: 1, Unrel: true}); err != nil {
			log.Print(err)
		}

		for {
			pkt, err := p2.Recv()
			if err != nil {
				if err == ErrClosed {
					msg := p2.Addr().String() + " disconnected"
					if p2.TimedOut() {
						msg += " (timed out)"
					}
					log.Print(msg)

					if !p2.IsSrv() {
						connectedPeersMu.Lock()
						connectedPeers--
						connectedPeersMu.Unlock()

						processLeave(p2.ID())
					}

					return
				}

				log.Print(err)
				continue
			}

			switch cmd := binary.BigEndian.Uint16(pkt.Data[0:2]); cmd {
			case 0x02:
				if pkt.Data[10]&AuthMechSRP > 0 {
					// Compute and send SRP_BYTES_A
					_, _, err := srp.NewClient([]byte(strings.ToLower(string(p.username))), passPhrase)
					if err != nil {
						log.Print(err)
						continue
					}

					A, a, err := srp.InitiateHandshake()
					if err != nil {
						log.Print(err)
						continue
					}

					p.srp_A = A
					p.srp_a = a

					data := make([]byte, 5+len(p.srp_A))
					data[0] = uint8(0x00)
					data[1] = uint8(0x51)
					binary.BigEndian.PutUint16(data[2:4], uint16(len(p.srp_A)))
					copy(data[4:4+len(p.srp_A)], p.srp_A)
					data[4+len(p.srp_A)] = uint8(1)

					ack, err := p2.Send(Pkt{Data: data, ChNo: 1})
					if err != nil {
						log.Print(err)
						continue
					}
					<-ack
				} else {
					// Compute and send s and v
					s, v, err := srp.NewClient([]byte(strings.ToLower(string(p.username))), passPhrase)
					if err != nil {
						log.Print(err)
						continue
					}

					data := make([]byte, 7+len(s)+len(v))
					data[0] = uint8(0x00)
					data[1] = uint8(0x50)
					binary.BigEndian.PutUint16(data[2:4], uint16(len(s)))
					copy(data[4:4+len(s)], s)
					binary.BigEndian.PutUint16(data[4+len(s) : 6+len(s)], uint16(len(v)))
					copy(data[6+len(s):6+len(s)+len(v)], v)
					data[6+len(s)+len(v)] = uint8(0)

					ack, err := p2.Send(Pkt{Data: data, ChNo: 1})
					if err != nil {
						log.Print(err)
						continue
					}
					<-ack
				}
			case 0x60:
				// Compute and send SRP_BYTES_M
				lenS := binary.BigEndian.Uint16(pkt.Data[2:4])
				s := pkt.Data[4 : lenS+4]
				B := pkt.Data[lenS+6:]

				K, err := srp.CompleteHandshake(p.srp_A, p.srp_a, []byte(strings.ToLower(string(p.username))), passPhrase, s, B)
				if err != nil {
					log.Print(err)
					continue
				}

				p.srp_K = K

				M := srp.CalculateM(p.username, s, p.srp_A, B, p.srp_K)

				data := make([]byte, 4+len(M))
				data[0] = uint8(0x00)
				data[1] = uint8(0x52)
				binary.BigEndian.PutUint16(data[2:4], uint16(len(M)))
				copy(data[4:], M)

				ack, err := p2.Send(Pkt{Data: data, ChNo: 1})
				if err != nil {
					log.Print(err)
					continue
				}
				<-ack
			case 0x0A:
				// Auth failed for some reason
				log.Print(ErrAuthFailed)

				data := []byte{
					uint8(0x00), uint8(0x0A),
					uint8(0x09), uint8(0x00), uint8(0x00), uint8(0x00), uint8(0x00),
				}

				ack, err := p.Send(Pkt{Data: data})
				if err != nil {
					log.Print(err)
				}
				<-ack

				p.SendDisco(0, true)
				p.Close()
				return
			case 0x03:
				// Auth succeeded
				ack, err := p2.Send(Pkt{Data: []byte{uint8(0), uint8(0x11), uint8(0), uint8(0)}, ChNo: 1})
				if err != nil {
					log.Print(err)
					continue
				}
				<-ack

				if !ignMedia {
					return
				}
			case 0x2A:
				// Definitions sent (by server)
				if !ignMedia {
					continue
				}

				v := []byte("5.4.0-dev-dd5a732fa")

				data := make([]byte, 8+len(v))
				copy(data[0:6], []byte{uint8(0), uint8(0x43), uint8(5), uint8(4), uint8(0), uint8(0)})
				binary.BigEndian.PutUint16(data[6:8], uint16(len(v)))
				copy(data[8:], v)

				ack, err := p2.Send(Pkt{Data: data, ChNo: 1})
				if err != nil {
					log.Print(err)
					continue
				}
				<-ack

				return
			}
		}
	} else {
		for {
			pkt, err := p2.Recv()
			if err != nil {
				if err == ErrClosed {
					msg := p2.Addr().String() + " disconnected"
					if p2.TimedOut() {
						msg += " (timed out)"
					}
					log.Print(msg)

					if !p2.IsSrv() {
						connectedPeersMu.Lock()
						connectedPeers--
						connectedPeersMu.Unlock()

						processLeave(p2.ID())
					}

					return
				}

				log.Print(err)
				continue
			}

			switch cmd := binary.BigEndian.Uint16(pkt.Data[0:2]); cmd {
			case 0x02:
				// Process data
				p2.username = pkt.Data[11:]

				// Send HELLO
				data := make([]byte, 13+len(p2.username))
				data[0] = uint8(0x00)
				data[1] = uint8(0x02)
				data[2] = uint8(0x1c)
				binary.BigEndian.PutUint16(data[3:5], uint16(0x0000))
				binary.BigEndian.PutUint16(data[5:7], uint16(0x0027))

				db, err := initAuthDB()
				if err != nil {
					log.Print(err)
					continue
				}

				pwd, err := readAuthItem(db, string(p2.username))
				if err != nil {
					log.Print(err)
					continue
				}

				db.Close()

				if pwd == "" {
					// New player
					p2.authMech = AuthMechFirstSRP

					binary.BigEndian.PutUint32(data[7:11], uint32(AuthMechFirstSRP))
				} else {
					// Existing player
					p2.authMech = AuthMechSRP

					binary.BigEndian.PutUint32(data[7:11], uint32(AuthMechSRP))
				}

				binary.BigEndian.PutUint16(data[11:13], uint16(len(p2.username)))
				copy(data[13:], p2.username)

				ack, err := p2.Send(Pkt{Data: data})
				if err != nil {
					log.Print(err)
					continue
				}
				<-ack
			case 0x50:
				// Process data
				// Make sure the client is allowed to use AuthMechFirstSRP
				if p2.authMech != AuthMechFirstSRP {
					log.Print(p2.Addr().String() + " used unsupported AuthMechFirstSRP")

					// Send ACCESS_DENIED
					data := []byte{
						uint8(0x00), uint8(0x0A),
						uint8(0x01), uint8(0x00), uint8(0x00), uint8(0x00), uint8(0x00),
					}

					ack, err := p2.Send(Pkt{Data: data})
					if err != nil {
						log.Print(err)
						continue
					}
					<-ack

					p2.SendDisco(0, true)
					p2.Close()
					return
				}

				// This is a new player, save verifier and salt
				lenS := binary.BigEndian.Uint16(pkt.Data[2:4])
				s := pkt.Data[4 : 4+lenS]

				lenV := binary.BigEndian.Uint16(pkt.Data[4+lenS : 6+lenS])
				v := pkt.Data[6+lenS : 6+lenS+lenV]

				pwd := encodeVerifierAndSalt(s, v)

				db, err := initAuthDB()
				if err != nil {
					log.Print(err)
					continue
				}

				err = addAuthItem(db, string(p2.username), pwd)
				if err != nil {
					log.Print(err)
					continue
				}

				err = addPrivItem(db, string(p2.username))
				if err != nil {
					log.Print(err)
					continue
				}

				db.Close()

				// Send AUTH_ACCEPT
				data := []byte{
					uint8(0x00), uint8(0x03),
					// Position stuff
					uint8(0x00), uint8(0x00), uint8(0x00), uint8(0x00),
					uint8(0x00), uint8(0x00), uint8(0x00), uint8(0x00),
					uint8(0x00), uint8(0x00), uint8(0x00), uint8(0x00),
					// Map seed
					uint8(0x00), uint8(0x00), uint8(0x00), uint8(0x00),
					uint8(0x00), uint8(0x00), uint8(0x00), uint8(0x00),
					// Send interval
					uint8(0x3D), uint8(0xB8), uint8(0x51), uint8(0xEC),
					// Sudo mode mechs
					uint8(0x00), uint8(0x00), uint8(0x00), uint8(0x02),
				}

				ack, err := p2.Send(Pkt{Data: data})
				if err != nil {
					log.Print(err)
					continue
				}
				<-ack

				// Connect to Minetest server
				fin2 := make(chan struct{}) // close-only
				Init(p2, p, ignMedia, fin2)
			case 0x51:
				// Process data
				// Make sure the client is allowed to use AuthMechSRP
				if p2.authMech != AuthMechSRP {
					log.Print(p2.Addr().String() + " used unsupported AuthMechSRP")

					// Send ACCESS_DENIED
					data := []byte{
						uint8(0x00), uint8(0x0A),
						uint8(0x01), uint8(0x00), uint8(0x00), uint8(0x00), uint8(0x00),
					}

					ack, err := p2.Send(Pkt{Data: data, ChNo: 0, Unrel: false})
					if err != nil {
						log.Print(err)
						continue
					}
					<-ack

					p2.SendDisco(0, true)
					p2.Close()
					return
				}

				lenA := binary.BigEndian.Uint16(pkt.Data[2:4])
				A := pkt.Data[4 : 4+lenA]

				db, err := initAuthDB()
				if err != nil {
					log.Print(err)
					continue
				}

				pwd, err := readAuthItem(db, string(p2.username))
				if err != nil {
					log.Print(err)
					continue
				}

				db.Close()

				s, v, err := decodeVerifierAndSalt(pwd)
				if err != nil {
					log.Print(err)
					continue
				}

				B, _, K, err := srp.Handshake(A, v)
				if err != nil {
					log.Print(err)
					continue
				}

				p2.srp_s = s
				p2.srp_A = A
				p2.srp_B = B
				p2.srp_K = K

				// Send SRP_BYTES_S_B
				data := make([]byte, 6+len(s)+len(B))
				data[0] = uint8(0x00)
				data[1] = uint8(0x60)
				binary.BigEndian.PutUint16(data[2:4], uint16(len(s)))
				copy(data[4:4+len(s)], s)
				binary.BigEndian.PutUint16(data[4+len(s):6+len(s)], uint16(len(B)))
				copy(data[6+len(s):6+len(s)+len(B)], B)

				ack, err := p2.Send(Pkt{Data: data})
				if err != nil {
					log.Print(err)
					continue
				}
				<-ack
			case 0x52:
				// Process data
				// Make sure the client is allowed to use AuthMechSRP
				if p2.authMech != AuthMechSRP {
					log.Print(p2.Addr().String() + " used unsupported AuthMechSRP")

					// Send ACCESS_DENIED
					data := []byte{
						uint8(0x00), uint8(0x0A),
						uint8(0x01), uint8(0x00), uint8(0x00), uint8(0x00), uint8(0x00),
					}

					ack, err := p2.Send(Pkt{Data: data})
					if err != nil {
						log.Print(err)
						continue
					}
					<-ack

					p2.SendDisco(0, true)
					p2.Close()
					return
				}

				lenM := binary.BigEndian.Uint16(pkt.Data[2:4])
				M := pkt.Data[4 : 4+lenM]

				M2 := srp.CalculateM(p2.username, p2.srp_s, p2.srp_A, p2.srp_B, p2.srp_K)

				if subtle.ConstantTimeCompare(M, M2) == 1 {
					// Password is correct
					// Send AUTH_ACCEPT
					data := []byte{
						uint8(0x00), uint8(0x03),
						// Position stuff
						uint8(0x00), uint8(0x00), uint8(0x00), uint8(0x00),
						uint8(0x00), uint8(0x00), uint8(0x00), uint8(0x00),
						uint8(0x00), uint8(0x00), uint8(0x00), uint8(0x00),
						// Map seed
						uint8(0x00), uint8(0x00), uint8(0x00), uint8(0x00),
						uint8(0x00), uint8(0x00), uint8(0x00), uint8(0x00),
						// Send interval
						uint8(0x3D), uint8(0xB8), uint8(0x51), uint8(0xEC),
						// Sudo mode mechs
						uint8(0x00), uint8(0x00), uint8(0x00), uint8(0x02),
					}

					ack, err := p2.Send(Pkt{Data: data})
					if err != nil {
						log.Print(err)
						continue
					}
					<-ack

					// Connect to Minetest server
					fin2 := make(chan struct{}) // close-only
					Init(p2, p, ignMedia, fin2)
				} else {
					// Client supplied wrong password
					log.Print("User " + string(p2.username) + " at " + p2.Addr().String() + " supplied wrong password")

					// Send ACCESS_DENIED
					data := []byte{
						uint8(0x00), uint8(0x0A),
						uint8(0x00), uint8(0x00), uint8(0x00), uint8(0x00), uint8(0x00),
					}

					ack, err := p2.Send(Pkt{Data: data})
					if err != nil {
						log.Print(err)
						continue
					}
					<-ack

					p2.SendDisco(0, true)
					p2.Close()
					return
				}
			case 0x11:
				return
			}
		}
	}
}

func init() {
	aoIDs = make(map[PeerID]map[uint16]bool)
	loadConfig()
}
