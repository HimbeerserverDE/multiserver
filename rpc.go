package main

import (
	"encoding/binary"
	"errors"
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
var rpcSrvs map[*Peer]struct{}

func (p *Peer) joinRpc() {
	data := make([]byte, 4+len(rpcCh))
	data[0] = uint8(0x00)
	data[1] = uint8(ToServerModChannelJoin)
	binary.BigEndian.PutUint16(data[2:4], uint16(len(rpcCh)))
	copy(data[4:], []byte(rpcCh))

	ack, err := p.Send(rudp.Pkt{Data: data})
	if err != nil {
		return
	}
	<-ack
}

func (p *Peer) leaveRpc() {
	data := make([]byte, 4+len(rpcCh))
	data[0] = uint8(0x00)
	data[1] = uint8(ToServerModChannelLeave)
	binary.BigEndian.PutUint16(data[2:4], uint16(len(rpcCh)))
	copy(data[4:], []byte(rpcCh))

	ack, err := p.Send(rudp.Pkt{Data: data})
	if err != nil {
		return
	}
	<-ack
}

func processRpc(p *Peer, pkt rudp.Pkt) bool {
	chlen := binary.BigEndian.Uint16(pkt.Data[2:4])
	ch := string(pkt.Data[4 : 4+chlen])
	senderlen := binary.BigEndian.Uint16(pkt.Data[4+chlen : 6+chlen])
	sender := string(pkt.Data[6+chlen : 6+chlen+senderlen])
	msglen := binary.BigEndian.Uint16(pkt.Data[6+chlen+senderlen : 8+chlen+senderlen])
	msg := string(pkt.Data[8+chlen+senderlen : 8+chlen+senderlen+msglen])

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
		go p.doRpc("->DEFSRV "+defsrv, rq)
	case "<-GETPEERCNT":
		cnt := strconv.Itoa(PeerCount())
		go p.doRpc("->PEERCNT "+cnt, rq)
	case "<-ISONLINE":
		online := "false"
		if IsOnline(strings.Join(strings.Split(msg, " ")[2:], " ")) {
			online = "true"
		}
		go p.doRpc("->ISONLINE "+online, rq)
	case "<-CHECKPRIVS":
		name := strings.Split(msg, " ")[2]
		privs := decodePrivs(strings.Join(strings.Split(msg, " ")[3:], " "))
		hasprivs := "false"
		if IsOnline(name) {
			has, err := PeerByUsername(name).CheckPrivs(privs)
			if err == nil && has {
				hasprivs = "true"
			}
		}
		go p.doRpc("->HASPRIVS "+hasprivs, rq)
	case "<-GETPRIVS":
		name := strings.Split(msg, " ")[2]
		var r string
		if IsOnline(name) {
			privs, err := PeerByUsername(name).Privs()
			if err == nil {
				r = strings.Replace(encodePrivs(privs), "|", ",", -1)
			}
		}
		go p.doRpc("->PRIVS "+r, rq)
	case "<-SETPRIVS":
		name := strings.Split(msg, " ")[2]
		privs := decodePrivs(strings.Join(strings.Split(msg, " ")[3:], " "))
		if IsOnline(name) {
			PeerByUsername(name).SetPrivs(privs)
		}
	case "<-GETSRV":
		name := strings.Split(msg, " ")[2]
		var srv string
		if IsOnline(name) {
			srv = PeerByUsername(name).ServerName()
		}
		go p.doRpc("->SRV "+srv, rq)
	case "<-REDIRECT":
		name := strings.Split(msg, " ")[2]
		tosrv := strings.Split(msg, " ")[3]
		if IsOnline(name) {
			go PeerByUsername(name).Redirect(tosrv)
		}
	case "<-GETADDR":
		name := strings.Split(msg, " ")[2]
		var addr string
		if IsOnline(name) {
			addr = PeerByUsername(name).Addr().String()
		}
		go p.doRpc("->ADDR "+addr, rq)
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

		go p.doRpc("->ISBANNED "+r, rq)
	case "<-BAN":
		target := strings.Split(msg, " ")[2]
		err := Ban(target)
		if err != nil {
			p2 := PeerByUsername(target)
			if p2 == nil {
				return true
			}

			p2.Ban()
		}
	case "<-UNBAN":
		target := strings.Split(msg, " ")[2]
		Unban(target)
	case "<-MT2MT":
		msg := strings.Join(strings.Split(msg, " ")[2:], " ")
		rpcSrvMu.Lock()
		for srv := range rpcSrvs {
			if srv.Addr().String() != p.Addr().String() {
				go srv.doRpc("->MT2MT "+msg, "--")
			}
		}
		rpcSrvMu.Unlock()
	}
	return true
}

func (p *Peer) doRpc(rpc, rq string) {
	if !p.UseRpc() {
		return
	}

	msg := rq + " " + rpc

	data := make([]byte, 6+len(rpcCh)+len(msg))
	data[0] = uint8(0x00)
	data[1] = uint8(ToServerModChannelMsg)
	binary.BigEndian.PutUint16(data[2:4], uint16(len(rpcCh)))
	copy(data[4:4+len(rpcCh)], []byte(rpcCh))
	binary.BigEndian.PutUint16(data[4+len(rpcCh):6+len(rpcCh)], uint16(len(msg)))
	copy(data[6+len(rpcCh):6+len(rpcCh)+len(msg)], []byte(msg))

	_, err := p.Send(rudp.Pkt{Data: data})
	if err != nil {
		return
	}
}

func connectRpc() {
	log.Print("Establishing RPC connections")

	servers := ConfKey("servers").(map[interface{}]interface{})
	for server := range servers {
		clt := &Peer{username: "rpc"}

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

		srv, err := Connect(conn, conn.RemoteAddr())
		if err != nil {
			log.Print(err)
			continue
		}

		fin := make(chan *Peer) // close-only
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

func handleRpc(srv *Peer) {
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

		switch cmd := binary.BigEndian.Uint16(pkt.Data[0:2]); cmd {
		case ToClientModChannelSignal:
			chlen := binary.BigEndian.Uint16(pkt.Data[3:5])
			ch := string(pkt.Data[5 : 5+chlen])
			if ch == rpcCh {
				switch sig := pkt.Data[2]; sig {
				case ModChSigJoinOk:
					srv.SetUseRpc(true)
				case ModChSigSetState:
					state := pkt.Data[5+chlen]
					if state == ModChStateRO {
						srv.SetUseRpc(false)
					}
				}
			}
		case ToClientModChannelMsg:
			processRpc(srv, pkt)
		}
	}
}

func OptimizeRPCConns() {
	rpcSrvMu.Lock()
	defer rpcSrvMu.Unlock()

ServerLoop:
	for p := range rpcSrvs {
		for _, p2 := range Peers() {
			if p2.Server() == nil {
				continue
			}
			if p2.Server().Addr().String() == p.Addr().String() {
				if p.NoClt() {
					p.SendDisco(0, true)
					p.Close()
				} else {
					p.SetUseRpc(false)
					p.leaveRpc()
				}

				delete(rpcSrvs, p)

				p3 := p2.Server()
				p3.SetUseRpc(true)
				p3.joinRpc()

				rpcSrvs[p3] = struct{}{}

				go func() {
					<-p3.Disco()
					rpcSrvMu.Lock()
					delete(rpcSrvs, p3)
					rpcSrvMu.Unlock()

					for p2.Server().Addr().String() == p3.Addr().String() {
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
		clt := &Peer{username: "rpc"}

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

		srv, err := Connect(conn, conn.RemoteAddr())
		if err != nil {
			log.Print(err)
			continue
		}

		fin := make(chan *Peer) // close-only
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
	rpcSrvs = make(map[*Peer]struct{})
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
