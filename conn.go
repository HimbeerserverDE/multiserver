package main

import (
	"bytes"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/anon55555/mt"
	"github.com/anon55555/mt/rudp"
)

var connectedConns int = 0
var connectedConnsMu sync.RWMutex

// A Conn is a connection to a client or server
type Conn struct {
	*rudp.Conn

	protoVer    uint16
	formspecVer uint16

	username string
	srp_s    []byte
	srp_A    []byte
	srp_a    []byte
	srp_B    []byte
	srp_K    []byte
	authMech int
	sudoMode bool

	stopforward bool
	forwardMu   sync.RWMutex

	redirectMu sync.Mutex
	srvMu      sync.RWMutex
	srv        *Conn

	initAoReceived   bool
	aoIDs            map[uint16]bool
	localPlayerCao   uint16
	currentPlayerCao uint16

	useRPCMu sync.RWMutex
	useRPC   bool
	noCLT    bool
	modChs   map[string]bool

	huds   map[uint32]bool
	sounds map[int32]bool
	blocks [][3]int16
	inv    *mt.Inv
}

// ProtoVer returns the protocol version of the Conn
func (c *Conn) ProtoVer() uint16 { return c.protoVer }

// FormspecVer returns the formspec API version of the Conn
func (c *Conn) FormspecVer() uint16 { return c.formspecVer + 1 }

// Addr returns the remote address of the Conn
func (c *Conn) Addr() net.Addr {
	return c.Conn.RemoteAddr()
}

// Username returns the username of the Conn
// if it isn't a server
func (c *Conn) Username() string { return c.username }

// Forward reports whether the Proxy func should continue or stop
func (c *Conn) Forward() bool {
	c.forwardMu.RLock()
	defer c.forwardMu.RUnlock()

	return !c.stopforward
}

// stopForwarding tells the Proxy func to stop
func (c *Conn) stopForwarding() {
	c.forwardMu.Lock()
	defer c.forwardMu.Unlock()

	c.stopforward = true
}

// Server returns the Conn this Conn is connected to
// if it isn't a server
func (c *Conn) Server() *Conn {
	c.srvMu.RLock()
	defer c.srvMu.RUnlock()

	return c.srv
}

// ServerName returns the name of the Conn this Conn is connected to
// if this Conn is not a server
func (c *Conn) ServerName() string {
	servers := ConfKey("servers").(map[interface{}]interface{})
	for server := range servers {
		if ConfKey("servers:"+server.(string)+":address") == c.Server().Addr().String() {
			return server.(string)
		}
	}

	return ""
}

// SetServer sets the Conn this Conn is connected to
// if this Conn is not a server
func (c *Conn) SetServer(s *Conn) {
	c.srvMu.Lock()
	defer c.srvMu.Unlock()

	c.srv = s
}

// UseRPC reports whether RPC messages can be sent to the Conn
func (c *Conn) UseRPC() bool {
	c.useRPCMu.RLock()
	defer c.useRPCMu.RUnlock()

	return c.useRPC
}

// SetUseRPC sets the value returned by UseRPC
func (c *Conn) SetUseRPC(useRPC bool) {
	c.useRPCMu.Lock()
	defer c.useRPCMu.Unlock()

	c.useRPC = useRPC
}

// NoCLT reports whether the Conn is RPC-only
func (c *Conn) NoCLT() bool { return c.noCLT }

// MakeRPCOnly marks the Conn as RPC-only
func (c *Conn) MakeRPCOnly() {
	c.noCLT = true
}

// Inv returns the inventory of the Conn
func (c *Conn) Inv() *mt.Inv { return c.inv }

// CloseWith denies access and disconnects the Conn
func (c *Conn) CloseWith(reason uint8, custom string, reconnect bool) error {
	defer c.Close()

	w := bytes.NewBuffer([]byte{0x00, ToClientAccessDenied})
	WriteUint8(w, reason)
	WriteBytes16(w, []byte(custom))
	if reconnect {
		WriteUint8(w, 1)
	} else {
		WriteUint8(w, 0)
	}

	_, err := c.Send(rudp.Pkt{Reader: w})
	if err != nil {
		return err
	}

	return nil
}

// Connect connects to the server on conn
// and closes conn when the Conn disconnects
func Connect(conn net.Conn) (*Conn, error) {
	srv := &Conn{Conn: rudp.Connect(conn)}

	ack, err := srv.Send(rudp.Pkt{Reader: bytes.NewReader([]byte{0, 0})})
	if err != nil {
		return nil, err
	}

	select {
	case <-time.After(8 * time.Second):
		srv.Close()

		return nil, fmt.Errorf("server at %s is unreachable", conn.RemoteAddr().String())
	case <-ack:
	}

	return srv, nil
}

// ConnCount reports how many client Conns are connected
func ConnCount() int {
	connectedConnsMu.RLock()
	defer connectedConnsMu.RUnlock()

	return connectedConns
}

// ConnsServer returns the client Conns that are connected to a server
func ConnsServer(server string) []*Conn {
	var r []*Conn
	for _, c := range Conns() {
		if c.ServerName() == server {
			r = append(r, c)
		}
	}
	return r
}
