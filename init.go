package main

import (
	"bytes"
	"crypto/subtle"
	"encoding/binary"
	"errors"
	"io"
	"log"
	"net"
	"strings"
	"time"

	"github.com/HimbeerserverDE/srp"
	"github.com/anon55555/mt/rudp"
)

// Init completes the initialisation of a connection to a server or client c2
func Init(c, c2 *Conn, ignMedia, noAccessDenied bool, fin chan *Conn) {
	defer close(fin)

	if c2.IsSrv() {
		// We're trying to connect to a server
		// INIT
		data := make([]byte, 11+len(c.Username()))
		data[0] = uint8(0x00)
		data[1] = uint8(ToServerInit)
		data[2] = uint8(0x1c)
		binary.BigEndian.PutUint16(data[3:5], uint16(0x0000))
		binary.BigEndian.PutUint16(data[5:7], uint16(ProtoMin))
		binary.BigEndian.PutUint16(data[7:9], uint16(ProtoLatest))
		binary.BigEndian.PutUint16(data[9:11], uint16(len(c.Username())))
		copy(data[11:], []byte(c.Username()))

		time.Sleep(250 * time.Millisecond)

		if _, err := c2.Send(rudp.Pkt{
			Reader: bytes.NewReader(data),
			PktInfo: rudp.PktInfo{
				Channel: 1,
				Unrel:   true,
			},
		}); err != nil {
			log.Print(err)
		}

		for {
			pkt, err := c2.Recv()
			if err != nil {
				if errors.Is(err, net.ErrClosed) {
					if err = c2.WhyClosed(); err != nil {
						log.Print(c2.Addr().String(), " disconnected with error: ", err)
					} else {
						log.Print(c2.Addr().String(), " disconnected")
					}

					return
				}

				log.Print(err)
				continue
			}

			r := ByteReader(pkt)

			switch cmd := ReadUint16(r); cmd {
			case ToClientHello:
				r.Seek(5, io.SeekStart)
				c2.protoVer = ReadUint16(r)
				r.Seek(10, io.SeekStart)
				authMech := ReadUint8(r)

				if authMech&AuthMechSRP > 0 {
					// Compute and send SRP_BYTES_A
					_, _, err := srp.NewClient([]byte(strings.ToLower(c.Username())), passPhrase)
					if err != nil {
						log.Print(err)
						continue
					}

					A, a, err := srp.InitiateHandshake()
					if err != nil {
						log.Print(err)
						continue
					}

					c.srp_A = A
					c.srp_a = a

					w := bytes.NewBuffer([]byte{0x00, ToServerSRPBytesA})
					WriteBytes16(w, c.srp_A)
					WriteUint8(w, 1)

					ack, err := c2.Send(rudp.Pkt{
						Reader: w,
						PktInfo: rudp.PktInfo{
							Channel: 1,
						},
					})

					if err != nil {
						log.Print(err)
						continue
					}
					<-ack
				} else {
					// Compute and send s and v
					s, v, err := srp.NewClient([]byte(strings.ToLower(c.Username())), passPhrase)
					if err != nil {
						log.Print(err)
						continue
					}

					w := bytes.NewBuffer([]byte{0x00, ToServerFirstSRP})
					WriteBytes16(w, s)
					WriteBytes16(w, v)
					WriteUint8(w, 0)

					ack, err := c2.Send(rudp.Pkt{
						Reader: w,
						PktInfo: rudp.PktInfo{
							Channel: 1,
						},
					})

					if err != nil {
						log.Print(err)
						continue
					}
					<-ack
				}
			case ToClientSrpBytesSB:
				// Compute and send SRP_BYTES_M
				s := ReadBytes16(r)
				r.Seek(2, io.SeekCurrent)
				B := make([]byte, r.Len())
				r.Read(B)

				K, err := srp.CompleteHandshake(c.srp_A, c.srp_a, []byte(strings.ToLower(c.Username())), passPhrase, s, B)
				if err != nil {
					log.Print(err)
					continue
				}

				c.srp_K = K

				M := srp.ClientProof([]byte(c.Username()), s, c.srp_A, B, c.srp_K)

				w := bytes.NewBuffer([]byte{0x00, ToServerSRPBytesM})
				WriteBytes16(w, M)

				ack, err := c2.Send(rudp.Pkt{
					Reader: w,
					PktInfo: rudp.PktInfo{
						Channel: 1,
					},
				})

				if err != nil {
					log.Print(err)
					continue
				}
				<-ack
			case ToClientAccessDenied:
				// Auth failed for some reason
				servers := ConfKey("servers").(map[interface{}]interface{})
				var srv string
				for server := range servers {
					if ConfKey("servers:"+server.(string)+":address") == c2.Addr().String() {
						srv = server.(string)
						break
					}
				}

				log.Print("access denied by server " + srv)

				if noAccessDenied {
					return
				}

				c.CloseWith(AccessDeniedServerFail, "", false)
				return
			case ToClientAuthAccept:
				// Auth succeeded
				defer func() {
					fin <- c2
				}()

				ack, err := c2.Send(rudp.Pkt{
					Reader: bytes.NewReader([]byte{0, ToServerInit2, 0, 0}),
					PktInfo: rudp.PktInfo{
						Channel: 1,
					},
				})

				if err != nil {
					log.Print(err)
					continue
				}
				<-ack

				if !ignMedia {
					return
				}
			case ToClientCSMRestrictionFlags:
				// Definitions sent (by server)
				if !ignMedia {
					continue
				}

				v := []byte("5.4.0-dev-dd5a732fa")

				w := bytes.NewBuffer([]byte{0x00, ToServerClientReady})
				w.Write([]byte{5, 4, 0, 0})
				WriteBytes16(w, v)

				_, err := c2.Send(rudp.Pkt{
					Reader: w,
					PktInfo: rudp.PktInfo{
						Channel: 1,
					},
				})

				if err != nil {
					log.Print(err)
					continue
				}

				return
			}
		}
	} else {
		// We're trying to initialize a client
		for {
			pkt, err := c2.Recv()
			if err != nil {
				if errors.Is(err, net.ErrClosed) {
					if err = c2.WhyClosed(); err != nil {
						log.Print(c2.Addr().String(), " disconnected with error: ", err)
					} else {
						log.Print(c2.Addr().String(), " disconnected")
					}

					connectedConnsMu.Lock()
					connectedConns--
					connectedConnsMu.Unlock()

					processLeave(c2)

					return
				}

				log.Print(err)
				continue
			}

			r := ByteReader(pkt)

			switch cmd := ReadUint16(r); cmd {
			case ToServerInit:
				// Process data
				r.Seek(9, io.SeekStart)
				c2.username = string(ReadBytes16(r))
				r.Seek(5, io.SeekStart)

				// Find protocol version
				cliProtoMin := ReadUint16(r)
				cliProtoMax := ReadUint16(r)

				var protov uint16
				if cliProtoMax >= ProtoMin || cliProtoMin <= ProtoLatest {
					if cliProtoMax > ProtoLatest {
						protov = ProtoLatest
					} else {
						protov = ProtoLatest
					}
				}

				c2.protoVer = protov

				if strict, ok := ConfKey("force_latest_proto").(bool); (ok && strict) && (protov != ProtoLatest) || protov < ProtoMin || protov > ProtoLatest {
					c2.CloseWith(AccessDeniedWrongVersion, "", false)
					fin <- c
					return
				}

				// Send HELLO
				data := make([]byte, 13+len(c2.Username()))
				data[0] = uint8(0x00)
				data[1] = uint8(ToClientHello)
				data[2] = uint8(0x1c)
				binary.BigEndian.PutUint16(data[3:5], uint16(0x0000))
				binary.BigEndian.PutUint16(data[5:7], uint16(protov))

				// Check if user is banned
				banned, bname, err := c2.IsBanned()
				if err != nil {
					log.Print(err)
					continue
				}

				if banned {
					log.Print("Banned user " + bname + " at " + c2.Addr().String() + " tried to connect")

					reason := "Your IP address is banned. Banned name is " + bname
					c2.CloseWith(AccessDeniedCustomString, reason, false)
					fin <- c
					return
				}

				// Check if user is already connected
				if IsOnline(c2.Username()) {
					c2.CloseWith(AccessDeniedAlreadyConnected, "", false)
					fin <- c
					return
				}

				// Check if username is reserved for media or RPC
				if c2.Username() == "media" || c2.Username() == "rpc" {
					c2.CloseWith(AccessDeniedWrongName, "", false)
					fin <- c
					return
				}

				db, err := initAuthDB()
				if err != nil {
					log.Print(err)
					continue
				}

				pwd, err := readAuthItem(db, c2.Username())
				if err != nil {
					log.Print(err)
					continue
				}

				db.Close()

				if pwd == "" {
					// New player
					c2.authMech = AuthMechFirstSRP
					binary.BigEndian.PutUint32(data[7:11], uint32(AuthMechFirstSRP))
				} else {
					// Existing player
					c2.authMech = AuthMechSRP
					binary.BigEndian.PutUint32(data[7:11], uint32(AuthMechSRP))
				}

				binary.BigEndian.PutUint16(data[11:13], uint16(len(c2.Username())))
				copy(data[13:], []byte(c2.Username()))

				ack, err := c2.Send(rudp.Pkt{Reader: bytes.NewReader(data)})
				if err != nil {
					log.Print(err)
					continue
				}
				<-ack
			case ToServerFirstSRP:
				// Process data
				// Make sure the client is allowed to use AuthMechFirstSRP
				if c2.authMech != AuthMechFirstSRP {
					log.Print(c2.Addr().String() + " used unsupported AuthMechFirstSRP")

					c2.CloseWith(AccessDeniedUnexpectedData, "", false)
					fin <- c
					return
				}

				// This is a new player, save verifier and salt
				s := ReadBytes16(r)
				v := ReadBytes16(r)

				empty := ReadUint8(r)

				// Also make sure to check for an empty password
				disallow, ok := ConfKey("disallow_empty_passwords").(bool)
				if ok && disallow && empty > 0 {
					log.Print(c2.Addr().String() + " used an empty password but disallow_empty_passwords is true")

					c2.CloseWith(AccessDeniedEmptyPassword, "", false)
					fin <- c
					return
				}

				pwd := encodeVerifierAndSalt(s, v)

				db, err := initAuthDB()
				if err != nil {
					log.Print(err)
					continue
				}

				err = addAuthItem(db, c2.Username(), pwd)
				if err != nil {
					log.Print(err)
					continue
				}

				err = addPrivItem(db, c2.Username())
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

				ack, err := c2.Send(rudp.Pkt{Reader: bytes.NewReader(data)})
				if err != nil {
					log.Print(err)
					continue
				}
				<-ack
			case ToServerSRPBytesA:
				// Process data
				// Make sure the client is allowed to use AuthMechSRP
				if c2.authMech != AuthMechSRP {
					log.Print(c2.Addr().String() + " used unsupported AuthMechSRP")

					c2.CloseWith(AccessDeniedUnexpectedData, "", false)
					return
				}

				A := ReadBytes16(r)

				db, err := initAuthDB()
				if err != nil {
					log.Print(err)
					continue
				}

				pwd, err := readAuthItem(db, c2.Username())
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

				c2.srp_s = s
				c2.srp_A = A
				c2.srp_B = B
				c2.srp_K = K

				// Send SRP_BYTES_S_B
				data := make([]byte, 6+len(s)+len(B))
				data[0] = uint8(0x00)
				data[1] = uint8(ToClientSrpBytesSB)
				binary.BigEndian.PutUint16(data[2:4], uint16(len(s)))
				copy(data[4:4+len(s)], s)
				binary.BigEndian.PutUint16(data[4+len(s):6+len(s)], uint16(len(B)))
				copy(data[6+len(s):6+len(s)+len(B)], B)

				w := bytes.NewBuffer([]byte{0x00, ToClientSrpBytesSB})
				WriteBytes16(w, s)
				WriteBytes16(w, B)

				ack, err := c2.Send(rudp.Pkt{Reader: w})
				if err != nil {
					log.Print(err)
					continue
				}
				<-ack
			case ToServerSRPBytesM:
				// Process data
				// Make sure the client is allowed to use AuthMechSRP
				if c2.authMech != AuthMechSRP {
					log.Print(c2.Addr().String() + " used unsupported AuthMechSRP")

					c2.CloseWith(AccessDeniedUnexpectedData, "", false)
					fin <- c
					return
				}

				M := ReadBytes16(r)
				M2 := srp.ClientProof([]byte(c2.Username()), c2.srp_s, c2.srp_A, c2.srp_B, c2.srp_K)

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

					ack, err := c2.Send(rudp.Pkt{Reader: bytes.NewReader(data)})
					if err != nil {
						log.Print(err)
						continue
					}
					<-ack
				} else {
					// Client supplied wrong password
					log.Print("User " + c2.Username() + " at " + c2.Addr().String() + " supplied wrong password")

					c2.CloseWith(AccessDeniedWrongPassword, "", false)
					fin <- c
					return
				}
			case ToServerInit2:
				c2.announceMedia()
			case ToServerRequestMedia:
				c2.sendMedia(r)
			case ToServerClientReady:
				// Second check if user is already connected
				// This is needed because the INIT packet
				// doesn't mark a player as online
				if IsOnline(c2.Username()) {
					c2.CloseWith(AccessDeniedAlreadyConnected, "", false)
					fin <- c
					return
				}

				defaultSrv := ConfKey("default_server").(string)

				defSrv := func() *Conn {
					defaultSrvAddr := ConfKey("servers:" + defaultSrv + ":address").(string)

					srvaddr, err := net.ResolveUDPAddr("udp", defaultSrvAddr)
					if err != nil {
						log.Print(err)
						return nil
					}

					conn, err := net.DialUDP("udp", nil, srvaddr)
					if err != nil {
						log.Print(err)
						return nil
					}

					srv, err := Connect(conn)
					if err != nil {
						log.Print(err)
						return nil
					}

					fin2 := make(chan *Conn) // close-only
					go Init(c2, srv, ignMedia, noAccessDenied, fin2)
					<-fin2

					go processJoin(c2)

					return srv
				}

				if forceDefaultServer, ok := ConfKey("force_default_server").(bool); !forceDefaultServer || !ok {
					srvname, err := StorageKey("server:" + c2.Username())
					if err != nil {
						srvname, ok = ConfKey("servers:" + ConfKey("default_server").(string) + ":address").(string)
						if !ok {
							go c2.SendChatMsg("Could not connect you to your last server!")

							fin <- defSrv()
							return
						}
					}

					straddr, ok := ConfKey("servers:" + srvname + ":address").(string)
					if !ok {
						go c2.SendChatMsg("Could not connect you to your last server!")

						fin <- defSrv()
						return
					}

					srvaddr, err := net.ResolveUDPAddr("udp", straddr)
					if err != nil {
						go c2.SendChatMsg("Could not connect you to your last server!")

						fin <- defSrv()
						return
					}

					conn, err := net.DialUDP("udp", nil, srvaddr)
					if err != nil {
						go c2.SendChatMsg("Could not connect you to your last server!")

						fin <- defSrv()
						return
					}

					if srvname != defaultSrv {
						srv, err := Connect(conn)
						if err != nil {
							go c2.SendChatMsg("Could not connect you to your last server!")

							fin <- defSrv()
							return
						}

						fin2 := make(chan *Conn) // close-only
						go Init(c2, srv, ignMedia, noAccessDenied, fin2)
						<-fin2

						go c2.updateDetachedInvs(srvname)
						go processJoin(c2)

						fin <- srv
						return
					}
				}

				fin <- defSrv()
				return
			}
		}
	}
}
