package main

import (
	"encoding/binary"
	"strconv"
	"strings"

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

func processRpc(src, dst *Peer, pkt rudp.Pkt) bool {
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
		defsrv, ok := GetConfKey("default_server").(string)
		if !ok {
			return true
		}
		src.doRpc("->DEFSRV "+defsrv, rq)
	case "<-GETPEERCNT":
		cnt := strconv.Itoa(GetPeerCount())
		src.doRpc("->PEERCNT "+cnt, rq)
	case "<-ISONLINE":
		online := "false"
		if IsOnline(strings.Join(strings.Split(msg, " ")[2:], " ")) {
			online = "true"
		}
		src.doRpc("->ISONLINE "+online, rq)
	case "<-CHECKPRIVS":
		name := strings.Split(msg, " ")[2]
		privs := decodePrivs(strings.Join(strings.Split(msg, " ")[3:], " "))
		hasprivs := "false"
		if IsOnline(name) {
			has, err := GetListener().GetPeerByUsername(name).CheckPrivs(privs)
			if err == nil && has {
				hasprivs = "true"
			}
		}
		src.doRpc("->HASPRIVS "+hasprivs, rq)
	case "<-GETSRV":
		name := strings.Split(msg, " ")[2]
		var srv string
		if IsOnline(name) {
			srv = GetListener().GetPeerByUsername(name).ServerName()
		}
		src.doRpc("->SRV "+srv, rq)
	case "<-REDIRECT":
		name := strings.Split(msg, " ")[2]
		tosrv := strings.Split(msg, " ")[3]
		if IsOnline(name) {
			go GetListener().GetPeerByUsername(name).Redirect(tosrv)
		}
	case "<-GETADDR":
		name := strings.Split(msg, " ")[2]
		var addr string
		if IsOnline(name) {
			addr = GetListener().GetPeerByUsername(name).Addr().String()
		}
		src.doRpc("->ADDR "+addr, rq)
	}
	return true
}

func (p *Peer) doRpc(rpc, rq string) {
	if !p.useRpc {
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

	ack, err := p.Send(rudp.Pkt{Data: data})
	if err != nil {
		return
	}
	<-ack
}
