package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/anon55555/mt/rudp"
)

const rpcCh = "multiserver"

const (
	ModChSigJoinOk = iota
	ModChSigJoinFail
	ModChSigLeaveOk
	ModChSigLeaveFail
	ModChSigChNotRegistered
	ModChSigSetState
)

const (
	ModChStateInit = iota
	ModChStateRW
	ModChStateRO
)

var rpcSrvMu sync.Mutex
var rpcSrvs map[*Conn]struct{}

func (c *Conn) joinRpc() {
	data := make([]byte, 4+len(rpcCh))
	data[0] = uint8(0x00)
	data[1] = uint8(ToServerModChannelJoin)
	binary.BigEndian.PutUint16(data[2:4], uint16(len(rpcCh)))
	copy(data[4:], []byte(rpcCh))

	ack, err := c.Send(rudp.Pkt{Reader: bytes.NewReader(data)})
	if err != nil {
		return
	}
	<-ack
}

func (c *Conn) leaveRpc() {
	data := make([]byte, 4+len(rpcCh))
	data[0] = uint8(0x00)
	data[1] = uint8(ToServerModChannelLeave)
	binary.BigEndian.PutUint16(data[2:4], uint16(len(rpcCh)))
	copy(data[4:], []byte(rpcCh))

	ack, err := c.Send(rudp.Pkt{Reader: bytes.NewReader(data)})
	if err != nil {
		return
	}
	<-ack
}

func processRpc(c *Conn, r *bytes.Reader) bool {
	ch := string(ReadBytes16(r))
	sender := string(ReadBytes16(r))
	msg := string(ReadBytes16(r))

	if ch != rpcCh || sender != "" {
		return false
	}

	rq := strings.Split(msg, " ")[0]

	switch cmd := strings.Split(msg, " ")[1]; cmd {
	case "<-ALERT":
		ChatSendAll(strings.Join(strings.Split(msg, " ")[2:], " "))
	case "<-GETDEFSRV":
		defsrv, ok := ConfKey("default_server").(string)
		if !ok {
			return true
		}
		go c.doRpc("->DEFSRV "+defsrv, rq)
	case "<-GETPEERCNT":
		cnt := strconv.Itoa(ConnCount())
		go c.doRpc("->PEERCNT "+cnt, rq)
	case "<-ISONLINE":
		online := "false"
		if IsOnline(strings.Join(strings.Split(msg, " ")[2:], " ")) {
			online = "true"
		}
		go c.doRpc("->ISONLINE "+online, rq)
	case "<-CHECKPRIVS":
		name := strings.Split(msg, " ")[2]
		privs := decodePrivs(strings.Join(strings.Split(msg, " ")[3:], " "))
		hasprivs := "false"

		has, err := CheckPrivs(name, privs)
		if err == nil && has {
			hasprivs = "true"
		}

		go c.doRpc("->HASPRIVS "+hasprivs, rq)
	case "<-GETPRIVS":
		name := strings.Split(msg, " ")[2]
		var r string

		privs, err := Privs(name)
		if err == nil {
			r = strings.Replace(encodePrivs(privs), "|", ",", -1)
		}

		go c.doRpc("->PRIVS "+r, rq)
	case "<-SETPRIVS":
		name := strings.Split(msg, " ")[2]
		privs := decodePrivs(strings.Join(strings.Split(msg, " ")[3:], " "))

		SetPrivs(name, privs)
	case "<-GETSRV":
		name := strings.Split(msg, " ")[2]
		var srv string
		if IsOnline(name) {
			srv = ConnByUsername(name).ServerName()
		}
		go c.doRpc("->SRV "+srv, rq)
	case "<-REDIRECT":
		name := strings.Split(msg, " ")[2]
		tosrv := strings.Split(msg, " ")[3]
		if IsOnline(name) {
			go ConnByUsername(name).Redirect(tosrv)
		}
	case "<-GETADDR":
		name := strings.Split(msg, " ")[2]
		var addr string
		if IsOnline(name) {
			addr = ConnByUsername(name).Addr().String()
		}
		go c.doRpc("->ADDR "+addr, rq)
	case "<-ISBANNED":
		db, err := initAuthDB()
		if err != nil {
			return true
		}
		defer db.Close()

		target := strings.Split(msg, " ")[2]

		if net.ParseIP(target) == nil {
			return true
		}

		name, err := readBanItem(db, target)
		if err != nil {
			return true
		}

		r := "false"
		if name != "" {
			r = "true"
		}

		go c.doRpc("->ISBANNED "+r, rq)
	case "<-BAN":
		target := strings.Split(msg, " ")[2]
		err := Ban(target, "not known")
		if err != nil {
			c2 := ConnByUsername(target)
			if c2 == nil {
				return true
			}

			c2.Ban()
		}
	case "<-UNBAN":
		target := strings.Split(msg, " ")[2]
		Unban(target)
	case "<-GETSRVS":
		var srvs string

		servers := ConfKey("servers").(map[interface{}]interface{})
		for server := range servers {
			srvs += server.(string) + ","
		}
		srvs = srvs[:len(srvs)-1]

		go c.doRpc("->SRVS "+srvs, rq)
	case "<-MT2MT":
		msg := strings.Join(strings.Split(msg, " ")[2:], " ")
		rpcSrvMu.Lock()
		for srv := range rpcSrvs {
			if srv.Addr().String() != c.Addr().String() {
				go srv.doRpc("->MT2MT true "+msg, "--")
			}
		}
		rpcSrvMu.Unlock()
	case "<-MSG2MT":
		tosrv := strings.Split(msg, " ")[2]
		addr, ok := ConfKey("servers:" + tosrv + ":address").(string)
		if !ok || addr == c.Addr().String() {
			return true
		}

		msg := strings.Join(strings.Split(msg, " ")[3:], " ")
		rpcSrvMu.Lock()
		for srv := range rpcSrvs {
			if srv.Addr().String() == addr {
				go srv.doRpc("->MT2MT false "+msg, "--")
			}
		}
		rpcSrvMu.Unlock()
	}
	return true
}

func (c *Conn) doRpc(rpc, rq string) {
	if !c.UseRpc() {
		return
	}

	msg := rq + " " + rpc

	w := bytes.NewBuffer([]byte{0x00, ToServerModChannelMsg})
	WriteBytes16(w, []byte(rpcCh))
	WriteBytes16(w, []byte(msg))

	_, err := c.Send(rudp.Pkt{Reader: w})
	if err != nil {
		return
	}
}

func connectRpc() {
	log.Print("Establishing RPC connections")

	servers := ConfKey("servers").(map[interface{}]interface{})
	for server := range servers {
		clt := &Conn{username: "rpc"}

		straddr := ConfKey("servers:" + server.(string) + ":address")

		srvaddr, err := net.ResolveUDPAddr("udp", straddr.(string))
		if err != nil {
			log.Print(err)
			continue
		}

		conn, err := net.DialUDP("udp", nil, srvaddr)
		if err != nil {
			log.Print(err)
			continue
		}

		srv, err := Connect(conn)
		if err != nil {
			log.Print(err)
			continue
		}

		fin := make(chan *Conn) // close-only
		go Init(clt, srv, true, true, fin)

		go func() {
			<-fin

			rpcSrvMu.Lock()
			rpcSrvs[srv] = struct{}{}
			rpcSrvMu.Unlock()

			go srv.joinRpc()
			go handleRpc(srv)
		}()
	}
}

func handleRpc(srv *Conn) {
	srv.MakeRpcOnly()
	for {
		pkt, err := srv.Recv()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				rpcSrvMu.Lock()
				delete(rpcSrvs, srv)
				rpcSrvMu.Unlock()
				break
			}

			log.Print(err)
			continue
		}

		r := ByteReader(pkt)

		switch cmd := ReadUint16(r); cmd {
		case ToClientModChannelSignal:
			r.Seek(1, io.SeekCurrent)

			ch := string(ReadBytes16(r))
			state := ReadUint8(r)

			if ch == rpcCh {
				r.Seek(2, io.SeekStart)

				switch sig := ReadUint8(r); sig {
				case ModChSigJoinOk:
					srv.SetUseRpc(true)
				case ModChSigSetState:
					if state == ModChStateRO {
						srv.SetUseRpc(false)
					}
				}
			}
		case ToClientModChannelMSG:
			processRpc(srv, r)
		}
	}
}

func OptimizeRPCConns() {
	rpcSrvMu.Lock()
	defer rpcSrvMu.Unlock()

ServerLoop:
	for c := range rpcSrvs {
		for _, c2 := range Conns() {
			if c2.Server() == nil {
				continue
			}
			if c2.Server().Addr().String() == c.Addr().String() {
				if c.NoClt() {
					c.Close()
				} else {
					c.SetUseRpc(false)
					c.leaveRpc()
				}

				delete(rpcSrvs, c)

				c3 := c2.Server()
				c3.SetUseRpc(true)
				c3.joinRpc()

				rpcSrvs[c3] = struct{}{}

				go func() {
					<-c3.Closed()
					rpcSrvMu.Lock()
					delete(rpcSrvs, c3)
					rpcSrvMu.Unlock()

					for c2.Server().Addr().String() == c3.Addr().String() {
					}
					OptimizeRPCConns()
				}()

				continue ServerLoop
			}
		}
	}

	go reconnectRpc(false)
}

func reconnectRpc(media bool) {
	servers := ConfKey("servers").(map[interface{}]interface{})
ServerLoop:
	for server := range servers {
		clt := &Conn{username: "rpc"}

		straddr := ConfKey("servers:" + server.(string) + ":address").(string)

		rpcSrvMu.Lock()
		for rpcsrv := range rpcSrvs {
			if rpcsrv.Addr().String() == straddr {
				rpcSrvMu.Unlock()
				continue ServerLoop
			}
		}
		rpcSrvMu.Unlock()

		// Also refetch media in case something has not
		// been downloaded yet
		if media {
			loadMedia(map[string]struct{}{server.(string): {}})
		}

		srvaddr, err := net.ResolveUDPAddr("udp", straddr)
		if err != nil {
			log.Print(err)
			continue
		}

		conn, err := net.DialUDP("udp", nil, srvaddr)
		if err != nil {
			log.Print(err)
			continue
		}

		srv, err := Connect(conn)
		if err != nil {
			log.Print(err)
			continue
		}

		fin := make(chan *Conn) // close-only
		go Init(clt, srv, true, true, fin)

		go func() {
			<-fin

			rpcSrvMu.Lock()
			rpcSrvs[srv] = struct{}{}
			rpcSrvMu.Unlock()

			go srv.joinRpc()
			go handleRpc(srv)
		}()
	}
}

func init() {
	rpcSrvMu.Lock()
	rpcSrvs = make(map[*Conn]struct{})
	rpcSrvMu.Unlock()

	reconnect, ok := ConfKey("server_reintegration_interval").(int)
	if !ok {
		reconnect = 600
	}

	connectRpc()

	go func() {
		reconnect := time.NewTicker(time.Duration(reconnect) * time.Second)
		for {
			select {
			case <-reconnect.C:
				log.Print("Reintegrating servers")
				reconnectRpc(true)
			}
		}
	}()
}
