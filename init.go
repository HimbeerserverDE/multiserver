package multiserver

import (
	"crypto/subtle"
	"encoding/binary"
	"errors"
	"log"
	"net"
	"strings"
	"time"

	"github.com/HimbeerserverDE/srp"
	"github.com/anon55555/mt/rudp"
)

var ErrAuthFailed = errors.New("authentication failure")

// Init authenticates to the server srv
// and finishes the initialisation process if ignMedia is true
func Init(p, p2 *Peer, ignMedia, noAccessDenied bool, fin chan *Peer) {
	defer close(fin)

	if p2.IsSrv() {
		// We're trying to connect to a server
		// INIT
		data := make([]byte, 11+len(p.username))
		data[0] = uint8(0x00)
		data[1] = uint8(ToServerInit)
		data[2] = uint8(0x1c)
		binary.BigEndian.PutUint16(data[3:5], uint16(0x0000))
		binary.BigEndian.PutUint16(data[5:7], uint16(0x0025))
		binary.BigEndian.PutUint16(data[7:9], uint16(0x0027))
		binary.BigEndian.PutUint16(data[9:11], uint16(len(p.username)))
		copy(data[11:], p.username)

		time.Sleep(250 * time.Millisecond)

		if _, err := p2.Send(rudp.Pkt{Data: data, ChNo: 1, Unrel: true}); err != nil {
			log.Print(err)
		}

		for {
			pkt, err := p2.Recv()
			if err != nil {
				if err == rudp.ErrClosed {
					msg := p2.Addr().String() + " disconnected"
					if p2.TimedOut() {
						msg += " (timed out)"
					}
					log.Print(msg)

					if !p2.IsSrv() {
						connectedPeersMu.Lock()
						connectedPeers--
						connectedPeersMu.Unlock()

						processLeave(p2)
					}

					return
				}

				log.Print(err)
				continue
			}

			switch cmd := binary.BigEndian.Uint16(pkt.Data[0:2]); cmd {
			case ToClientHello:
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
					data[1] = uint8(ToServerSrpBytesA)
					binary.BigEndian.PutUint16(data[2:4], uint16(len(p.srp_A)))
					copy(data[4:4+len(p.srp_A)], p.srp_A)
					data[4+len(p.srp_A)] = uint8(1)

					ack, err := p2.Send(rudp.Pkt{Data: data, ChNo: 1})
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
					data[1] = uint8(ToServerFirstSrp)
					binary.BigEndian.PutUint16(data[2:4], uint16(len(s)))
					copy(data[4:4+len(s)], s)
					binary.BigEndian.PutUint16(data[4+len(s):6+len(s)], uint16(len(v)))
					copy(data[6+len(s):6+len(s)+len(v)], v)
					data[6+len(s)+len(v)] = uint8(0)

					ack, err := p2.Send(rudp.Pkt{Data: data, ChNo: 1})
					if err != nil {
						log.Print(err)
						continue
					}
					<-ack
				}
			case ToClientSrpBytesSB:
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
				data[1] = uint8(ToServerSrpBytesM)
				binary.BigEndian.PutUint16(data[2:4], uint16(len(M)))
				copy(data[4:], M)

				ack, err := p2.Send(rudp.Pkt{Data: data, ChNo: 1})
				if err != nil {
					log.Print(err)
					continue
				}
				<-ack
			case ToClientAccessDenied:
				// Auth failed for some reason
				log.Print(ErrAuthFailed)

				if noAccessDenied {
					return
				}

				data := []byte{
					0, ToClientAccessDenied,
					AccessDeniedServerFail, 0, 0, 0, 0,
				}

				ack, err := p.Send(rudp.Pkt{Data: data})
				if err != nil {
					log.Print(err)
				}
				<-ack

				p.SendDisco(0, true)
				p.Close()
				return
			case ToClientAuthAccept:
				// Auth succeeded
				ack, err := p2.Send(rudp.Pkt{Data: []byte{0, ToServerInit2, 0, 0}, ChNo: 1})
				if err != nil {
					log.Print(err)
					continue
				}
				<-ack

				if !ignMedia {
					return
				}
			case ToClientCsmRestrictionFlags:
				// Definitions sent (by server)
				if !ignMedia {
					continue
				}

				v := []byte("5.4.0-dev-dd5a732fa")

				data := make([]byte, 8+len(v))
				copy(data[0:6], []byte{uint8(0), uint8(ToServerClientReady), uint8(5), uint8(4), uint8(0), uint8(0)})
				binary.BigEndian.PutUint16(data[6:8], uint16(len(v)))
				copy(data[8:], v)

				ack, err := p2.Send(rudp.Pkt{Data: data, ChNo: 1})
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
				if err == rudp.ErrClosed {
					msg := p2.Addr().String() + " disconnected"
					if p2.TimedOut() {
						msg += " (timed out)"
					}
					log.Print(msg)

					if !p2.IsSrv() {
						connectedPeersMu.Lock()
						connectedPeers--
						connectedPeersMu.Unlock()

						processLeave(p2)
					}

					return
				}

				log.Print(err)
				continue
			}

			switch cmd := binary.BigEndian.Uint16(pkt.Data[0:2]); cmd {
			case ToServerInit:
				// Process data
				p2.username = pkt.Data[11:]

				// Send HELLO
				data := make([]byte, 13+len(p2.username))
				data[0] = uint8(0x00)
				data[1] = uint8(ToClientHello)
				data[2] = uint8(0x1c)
				binary.BigEndian.PutUint16(data[3:5], uint16(0x0000))
				binary.BigEndian.PutUint16(data[5:7], uint16(0x0027))

				// Check if user is already connected
				if IsOnline(string(p2.username)) {
					data := []byte{
						0, ToClientAccessDenied,
						AccessDeniedAlreadyConnected, 0, 0, 0, 0,
					}

					ack, err := p2.Send(rudp.Pkt{Data: data})
					if err != nil {
						log.Print(err)
						continue
					}
					<-ack

					p2.SendDisco(0, true)
					p2.Close()
					fin <- p
					return
				}

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

				ack, err := p2.Send(rudp.Pkt{Data: data})
				if err != nil {
					log.Print(err)
					continue
				}
				<-ack
			case ToServerFirstSrp:
				// Process data
				// Make sure the client is allowed to use AuthMechFirstSRP
				if p2.authMech != AuthMechFirstSRP {
					log.Print(p2.Addr().String() + " used unsupported AuthMechFirstSRP")

					// Send ACCESS_DENIED
					data := []byte{
						0, ToClientAccessDenied,
						AccessDeniedUnexpectedData, 0, 0, 0, 0,
					}

					ack, err := p2.Send(rudp.Pkt{Data: data})
					if err != nil {
						log.Print(err)
						continue
					}
					<-ack

					p2.SendDisco(0, true)
					p2.Close()
					fin <- p
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
					0, ToClientAuthAccept,
					// Position stuff
					0, 0, 0, 0,
					0, 0, 0, 0,
					0, 0, 0, 0,
					// Map seed
					0, 0, 0, 0,
					0, 0, 0, 0,
					// Send interval
					0x3D, 0xB8, 0x51, 0xEC,
					// Sudo mode mechs
					0, 0, 0, AuthMechSRP,
				}

				ack, err := p2.Send(rudp.Pkt{Data: data})
				if err != nil {
					log.Print(err)
					continue
				}
				<-ack
			case ToServerSrpBytesA:
				// Process data
				// Make sure the client is allowed to use AuthMechSRP
				if p2.authMech != AuthMechSRP {
					log.Print(p2.Addr().String() + " used unsupported AuthMechSRP")

					// Send ACCESS_DENIED
					data := []byte{
						0, ToClientAccessDenied,
						AccessDeniedUnexpectedData, 0, 0, 0, 0,
					}

					ack, err := p2.Send(rudp.Pkt{Data: data, ChNo: 0, Unrel: false})
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
				data[1] = uint8(ToClientSrpBytesSB)
				binary.BigEndian.PutUint16(data[2:4], uint16(len(s)))
				copy(data[4:4+len(s)], s)
				binary.BigEndian.PutUint16(data[4+len(s):6+len(s)], uint16(len(B)))
				copy(data[6+len(s):6+len(s)+len(B)], B)

				ack, err := p2.Send(rudp.Pkt{Data: data})
				if err != nil {
					log.Print(err)
					continue
				}
				<-ack
			case ToServerSrpBytesM:
				// Process data
				// Make sure the client is allowed to use AuthMechSRP
				if p2.authMech != AuthMechSRP {
					log.Print(p2.Addr().String() + " used unsupported AuthMechSRP")

					// Send ACCESS_DENIED
					data := []byte{
						0, ToClientAccessDenied,
						AccessDeniedUnexpectedData, 0, 0, 0, 0,
					}

					ack, err := p2.Send(rudp.Pkt{Data: data})
					if err != nil {
						log.Print(err)
						continue
					}
					<-ack

					p2.SendDisco(0, true)
					p2.Close()
					fin <- p
					return
				}

				lenM := binary.BigEndian.Uint16(pkt.Data[2:4])
				M := pkt.Data[4 : 4+lenM]

				M2 := srp.CalculateM(p2.username, p2.srp_s, p2.srp_A, p2.srp_B, p2.srp_K)

				if subtle.ConstantTimeCompare(M, M2) == 1 {
					// Password is correct
					// Send AUTH_ACCEPT
					data := []byte{
						0, ToClientAuthAccept,
						// Position stuff
						0, 0, 0, 0,
						0, 0, 0, 0,
						0, 0, 0, 0,
						// Map seed
						0, 0, 0, 0,
						0, 0, 0, 0,
						// Send interval
						0x3D, 0xB8, 0x51, 0xEC,
						// Sudo mode mechs
						0, 0, 0, AuthMechSRP,
					}

					ack, err := p2.Send(rudp.Pkt{Data: data})
					if err != nil {
						log.Print(err)
						continue
					}
					<-ack
				} else {
					// Client supplied wrong password
					log.Print("User " + string(p2.username) + " at " + p2.Addr().String() + " supplied wrong password")

					// Send ACCESS_DENIED
					data := []byte{
						0, ToClientAccessDenied,
						AccessDeniedWrongPassword, 0, 0, 0, 0,
					}

					ack, err := p2.Send(rudp.Pkt{Data: data})
					if err != nil {
						log.Print(err)
						continue
					}
					<-ack

					p2.SendDisco(0, true)
					p2.Close()
					fin <- p
					return
				}
			case ToServerInit2:
				p2.announceMedia()
			case ToServerRequestMedia:
				p2.sendMedia(pkt.Data[2:])
			case ToServerClientReady:
				if forceDefaultServer, ok := GetConfKey("force_default_server").(bool); !forceDefaultServer || !ok {
					srvname, err := GetStorageKey("server:" + string(p2.username))
					if err != nil {
						srvname, ok = GetConfKey("servers:" + GetConfKey("default_server").(string) + ":address").(string)
						if !ok {
							go p2.SendChatMsg("Could not connect you to your last server!")

							fin2 := make(chan *Peer) // close-only
							Init(p2, p, ignMedia, noAccessDenied, fin2)
							go processJoin(p2)

							fin <- p
							return
						}
					}

					straddr, ok := GetConfKey("servers:" + srvname + ":address").(string)
					if !ok {
						go p2.SendChatMsg("Could not connect you to your last server!")

						fin2 := make(chan *Peer) // close-only
						Init(p2, p, ignMedia, noAccessDenied, fin2)
						go processJoin(p2)

						fin <- p
						return
					}

					srvaddr, err := net.ResolveUDPAddr("udp", straddr)
					if err != nil {
						go p2.SendChatMsg("Could not connect you to your last server!")

						fin2 := make(chan *Peer) // close-only
						Init(p2, p, ignMedia, noAccessDenied, fin2)
						go processJoin(p2)

						fin <- p
						return
					}

					conn, err := net.DialUDP("udp", nil, srvaddr)
					if err != nil {
						go p2.SendChatMsg("Could not connect you to your last server!")

						fin2 := make(chan *Peer) // close-only
						Init(p2, p, ignMedia, noAccessDenied, fin2)
						go processJoin(p2)

						fin <- p
						return
					}

					if straddr != p.Addr().String() {
						srv, err := Connect(conn, conn.RemoteAddr())
						if err != nil {
							go p2.SendChatMsg("Could not connect you to your last server!")

							fin2 := make(chan *Peer) // close-only
							Init(p2, p, ignMedia, noAccessDenied, fin2)
							go processJoin(p2)

							fin <- p
							return
						}

						fin2 := make(chan *Peer) // close-only
						Init(p2, srv, ignMedia, noAccessDenied, fin2)
						go p2.updateDetachedInvs(srvname)
						go processJoin(p2)

						fin <- srv
						return
					}
				}

				fin2 := make(chan *Peer) // close-only
				Init(p2, p, ignMedia, noAccessDenied, fin2)
				go processJoin(p2)

				fin <- p
				return
			}
		}
	}
}
