package multiserver

import (
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/HimbeerserverDE/srp"
	_ "github.com/mattn/go-sqlite3"
)

var ErrAuthFailed = errors.New("authentication failure")
var ErrServerDoesNotExist = errors.New("server doesn't exist")
var ErrAlreadyConnected = errors.New("already connected to server")
var ErrServerUnreachable = errors.New("server is unreachable")

var passPhrase []byte = []byte("jK7BPRoxM9ffwh7Z")

var connectedPeers int = 0

const (
	// ConnTimeout is the amount of time after no packets being received
	// from a Peer that it is automatically disconnected
	ConnTimeout = 30 * time.Second

	// PingTimeout is the amount of time after no packets being sent
	// to a Peer that a CtlPing is automatically sent to prevent timeout
	PingTimeout = 5 * time.Second
)

const (
	AuthMechSRP      = 0x00000002
	AuthMechFirstSRP = 0x00000004
)

// A Peer is a connection to a client or server
type Peer struct {
	conn net.PacketConn
	addr net.Addr

	disco chan struct{} // close-only

	id PeerID

	pkts     chan Pkt
	errs     chan error    // don't close
	timedout chan struct{} // close only

	chans [ChannelCount]pktchan // read/write

	mu       sync.RWMutex
	idOfPeer PeerID
	timeout  *time.Timer
	ping     *time.Ticker

	username []byte

	srp_s []byte
	srp_A []byte
	srp_a []byte
	srp_B []byte
	srp_K []byte

	authMech int

	forward bool

	srv *Peer

	initAoReceived bool
}

type pktchan struct {
	insplit map[seqnum][][]byte
	inrel   map[seqnum][]byte
	inrelsn seqnum

	ackchans sync.Map // map[seqnum]chan struct{}

	outsplitmu sync.Mutex
	outsplitsn seqnum

	outrelmu  sync.Mutex
	outrelsn  seqnum
	outrelwin seqnum
}

// Conn returns the net.PacketConn used to communicate with the Peer
func (p *Peer) Conn() net.PacketConn { return p.conn }

// Addr returns the address of the peer
func (p *Peer) Addr() net.Addr { return p.addr }

// Disco returns a channel that is closed when the Peer is closed
func (p *Peer) Disco() <-chan struct{} { return p.disco }

// ID returns the ID of the peer
func (p *Peer) ID() PeerID { return p.id }

// IsSrv reports whether the Peer is a server
func (p *Peer) IsSrv() bool {
	return p.ID() == PeerIDSrv
}

// TimedOut reports whether the Peer has timed out
func (p *Peer) TimedOut() bool {
	select {
	case <-p.timedout:
		return true
	default:
		return false
	}
}

// Forward reports whether the Proxy func should continue or stop
func (p *Peer) Forward() bool { return p.forward }

// StopForwarding tells the Proxy func to stop
func (p *Peer) StopForwarding() { p.forward = false }

// StartForwarding makes forwarding possible again
func (p *Peer) StartForwarding() { p.forward = true }

// Server returns the Peer this Peer is connected to
// if this Peer is not a server
func (p *Peer) Server() *Peer { return p.srv }

// SetServer sets the Peer this Peer is connected to
// if this Peer is not a server
func (p *Peer) SetServer(s *Peer) { p.srv = s }

// Recv receives a packet from the Peer
// You should keep calling this until it returns ErrClosed
// so it doesn't leak a goroutine
func (p *Peer) Recv() (Pkt, error) {
	select {
	case pkt, ok := <-p.pkts:
		if !ok {
			select {
			case err := <-p.errs:
				return Pkt{}, err
			default:
				return Pkt{}, ErrClosed
			}
		}
		return pkt, nil
	case err := <-p.errs:
		return Pkt{}, err
	}
}

// Close closes the Peer but does not send a disconnect packet
func (p *Peer) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	select {
	case <-p.Disco():
		return ErrClosed
	default:
	}

	p.timeout.Stop()
	p.timeout = nil
	p.ping.Stop()
	p.ping = nil

	close(p.disco)

	return nil
}

func newPeer(conn net.PacketConn, addr net.Addr, id, idOfPeer PeerID) *Peer {
	p := &Peer{
		conn:     conn,
		addr:     addr,
		id:       id,
		idOfPeer: idOfPeer,

		pkts:  make(chan Pkt),
		disco: make(chan struct{}),
		errs:  make(chan error),
	}

	for i := range p.chans {
		p.chans[i] = pktchan{
			insplit: make(map[seqnum][][]byte),
			inrel:   make(map[seqnum][]byte),
			inrelsn: seqnumInit,

			outsplitsn: seqnumInit,
			outrelsn:   seqnumInit,
			outrelwin:  seqnumInit,
		}
	}

	p.timedout = make(chan struct{})
	p.timeout = time.AfterFunc(ConnTimeout, func() {
		close(p.timedout)

		p.SendDisco(0, true)
		p.Close()
	})

	p.ping = time.NewTicker(PingTimeout)
	go p.sendPings(p.ping.C)

	p.forward = true

	if !p.IsSrv() {
		aoIDs[p.ID()] = make(map[uint16]bool)
	}

	return p
}

func (p *Peer) sendPings(ping <-chan time.Time) {
	pkt := rawPkt{Data: []byte{uint8(rawTypeCtl), uint8(ctlPing)}}

	for {
		select {
		case <-ping:
			if _, err := p.sendRaw(pkt); err != nil {
				p.errs <- fmt.Errorf("can't send ping: %w", err)
			}
		case <-p.Disco():
			return
		}
	}
}

// Connect connects to the server on conn
// and closes conn when the Peer disconnects
func Connect(conn net.PacketConn, addr net.Addr) *Peer {
	srv := newPeer(conn, addr, PeerIDSrv, PeerIDNil)

	pkts := make(chan netPkt)
	go readNetPkts(conn, pkts, srv.errs)
	go srv.processNetPkts(pkts)

	ack, err := srv.Send(Pkt{Data: []byte{uint8(0), uint8(0)}, ChNo: 0, Unrel: false})
	if err != nil {
		log.Print(err)
	}

	t := time.Now()
	for time.Since(t).Seconds() < 8 {
		breakloop := false

		select {
		case <-ack:
			breakloop = true
		default:
		}

		if breakloop {
			break
		}
	}
	if time.Since(t).Seconds() >= 8 {
		srv.SendDisco(0, true)
		srv.Close()

		conn.Close()

		return nil
	}

	srv.sendAck(0, true, 65500)

	go func() {
		<-srv.Disco()
		conn.Close()
	}()

	return srv
}

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
						connectedPeers--
						processLeave(p2.ID())
					}

					return
				}

				log.Print(err)
				continue
			}

			switch cmd := binary.BigEndian.Uint16(pkt.Data[0:2]); cmd {
			case 0x02:
				if pkt.Data[10]&2 > 0 {
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

					ack, err := p2.Send(Pkt{Data: data, ChNo: 1, Unrel: false})
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
					binary.BigEndian.PutUint16(data[4+len(s):6+len(s)], uint16(len(v)))
					copy(data[6+len(s):6+len(s)+len(v)], v)
					data[6+len(s)+len(v)] = uint8(0)

					ack, err := p2.Send(Pkt{Data: data, ChNo: 1, Unrel: false})
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

				ack, err := p2.Send(Pkt{Data: data, ChNo: 1, Unrel: false})
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

				ack, err := p.Send(Pkt{Data: data, ChNo: 0, Unrel: false})
				if err != nil {
					log.Print(err)
				}
				<-ack

				p.SendDisco(0, true)
				p.Close()
				return
			case 0x03:
				// Auth succeeded
				ack, err := p2.Send(Pkt{Data: []byte{uint8(0), uint8(0x11), uint8(0), uint8(0)}, ChNo: 1, Unrel: false})
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

				ack, err := p2.Send(Pkt{Data: data, ChNo: 1, Unrel: false})
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
						connectedPeers--
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

				// Lua Callback
				processJoin(p2.ID())

				// Send HELLO
				data := make([]byte, 13+len(p2.username))
				data[0] = uint8(0x00)
				data[1] = uint8(0x02)
				data[2] = uint8(0x1c)
				binary.BigEndian.PutUint16(data[3:5], uint16(0x0000))
				binary.BigEndian.PutUint16(data[5:7], uint16(0x0027))

				db, err := initDB()
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

				ack, err := p2.Send(Pkt{Data: data, ChNo: 0, Unrel: false})
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

				// This is a new player, save verifier and salt
				lenS := binary.BigEndian.Uint16(pkt.Data[2:4])
				s := pkt.Data[4 : 4+lenS]

				lenV := binary.BigEndian.Uint16(pkt.Data[4+lenS : 6+lenS])
				v := pkt.Data[6+lenS : 6+lenS+lenV]

				pwd := encodeVerifierAndSalt(s, v)

				db, err := initDB()
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

				ack, err := p2.Send(Pkt{Data: data, ChNo: 0, Unrel: false})
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

				db, err := initDB()
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

				ack, err := p2.Send(Pkt{Data: data, ChNo: 0, Unrel: false})
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

					ack, err := p2.Send(Pkt{Data: data, ChNo: 0, Unrel: false})
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
			case 0x11:
				return
			}
		}
	}
}

// Redirect closes the connection to srv1
// and redirects the client to srv2
func (p *Peer) Redirect(newsrv string) error {
	defer processRedirectDone(p, newsrv)

	straddr := GetConfKey("servers:" + newsrv + ":address")
	if straddr == nil || fmt.Sprintf("%T", straddr) != "string" {
		return ErrServerDoesNotExist
	}

	if p.Server().Addr().String() == straddr {
		return ErrAlreadyConnected
	}

	srvaddr, err := net.ResolveUDPAddr("udp", straddr.(string))
	if err != nil {
		return err
	}

	conn, err := net.DialUDP("udp", nil, srvaddr)
	if err != nil {
		return err
	}
	srv := Connect(conn, conn.RemoteAddr())

	if srv == nil {
		return ErrServerUnreachable
	}

	// Remove active objects
	len := 0
	for _ = range aoIDs[p.ID()] {
		len++
	}

	data := make([]byte, 6+len*2)
	data[0] = uint8(0x00)
	data[1] = uint8(0x31)
	binary.BigEndian.PutUint16(data[2:4], uint16(len))
	i := 4
	for ao := range aoIDs[p.ID()] {
		binary.BigEndian.PutUint16(data[i:2+i], ao)

		i += 2
	}
	binary.BigEndian.PutUint16(data[i:2+i], uint16(0))

	ack, err := p.Send(Pkt{Data: data, ChNo: 0, Unrel: false})
	if err != nil {
		return err
	}
	<-ack

	aoIDs[p.ID()] = make(map[uint16]bool)
	p.initAoReceived = false

	p.Server().StopForwarding()

	fin := make(chan struct{}) // close-only
	go Init(p, srv, true, fin)
	<-fin

	p.SetServer(srv)

	go Proxy(p, srv)
	go Proxy(srv, p)

	log.Print(p.Addr().String() + " redirected to " + newsrv)

	return nil
}

// encodeVerifierAndSalt encodes SRP verifier and salt into DB-ready string
func encodeVerifierAndSalt(s, v []byte) string {
	return base64.StdEncoding.EncodeToString(s) + "#" + base64.StdEncoding.EncodeToString(v)
}

// decodeVerifierAndSalt decodes DB-ready string into SRP verifier and salt
func decodeVerifierAndSalt(src string) ([]byte, []byte, error) {
	sString := strings.Split(src, "#")[0]
	vString := strings.Split(src, "#")[1]

	s, err := base64.StdEncoding.DecodeString(sString)
	if err != nil {
		return nil, nil, err
	}

	v, err := base64.StdEncoding.DecodeString(vString)
	if err != nil {
		return nil, nil, err
	}

	return s, v, nil
}

// initDB opens auth.sqlite and creates the required tables
// if they don't exist
// It returns said database
func initDB() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", "storage/auth.sqlite")
	if err != nil {
		return nil, err
	}
	if db == nil {
		panic("DB is nil")
	}

	sql_table := `CREATE TABLE IF NOT EXISTS auth (
		name VARCHAR(32) NOT NULL,
		password VARCHAR(512) NOT NULL
	);
	CREATE TABLE IF NOT EXISTS privileges (
		name VARCHAR(32) NOT NULL,
		privileges VARCHAR(1024)
	);
	`

	_, err = db.Exec(sql_table)
	if err != nil {
		return nil, err
	}

	return db, nil
}

// addAuthItem inserts an auth DB entry
func addAuthItem(db *sql.DB, name, password string) error {
	sql_addAuthItem := `INSERT INTO auth (
		name,
		password
	) VALUES (
		?,
		?
	);
	`

	stmt, err := db.Prepare(sql_addAuthItem)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(name, password)
	if err != nil {
		return err
	}

	return nil
}

// modAuthItem updates an auth DB entry
func modAuthItem(db *sql.DB, name, password string) error {
	sql_modAuthItem := `UPDATE auth SET password = ? WHERE name = ?;`

	stmt, err := db.Prepare(sql_modAuthItem)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(password, name)
	if err != nil {
		return err
	}

	return nil
}

// readAuthItem selects and reads an auth DB entry
func readAuthItem(db *sql.DB, name string) (string, error) {
	sql_readAuthItem := `SELECT password FROM auth WHERE name = ?;`

	stmt, err := db.Prepare(sql_readAuthItem)
	if err != nil {
		return "", err
	}
	defer stmt.Close()

	rows, err := stmt.Query(name)
	if err != nil {
		return "", err
	}

	var r string

	for rows.Next() {
		err = rows.Scan(&r)
	}

	return r, nil
}

func GetPeerCount() int {
	return connectedPeers
}
