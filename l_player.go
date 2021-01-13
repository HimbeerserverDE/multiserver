package multiserver

import (
	"encoding/binary"
	"github.com/yuin/gopher-lua"
	"log"
)

var joinHandlers []*lua.LFunction
var leaveHandlers []*lua.LFunction

func registerOnJoinPlayer(L *lua.LState) int {
	f := L.ToFunction(1)
	joinHandlers = append(joinHandlers, f)
	return 0
}

func registerOnLeavePlayer(L *lua.LState) int {
	f := L.ToFunction(1)
	leaveHandlers = append(leaveHandlers, f)
	return 0
}

func processJoin(peerid PeerID) {
	for i := range joinHandlers {
		if err := l.CallByParam(lua.P{Fn: joinHandlers[i], NRet: 0, Protect: true}, lua.LNumber(peerid)); err != nil {
			log.Print(err)
			End(true, true)
		}
	}
}

func processLeave(peerid PeerID) {
	for i := range leaveHandlers {
		if err := l.CallByParam(lua.P{Fn: leaveHandlers[i], NRet: 0, Protect: true}, lua.LNumber(peerid)); err != nil {
			log.Print(err)
			End(true, true)
		}
	}
}

func getPlayerName(L *lua.LState) int {
	id := L.ToInt(1)
	l := GetListener()
	p := l.GetPeerByID(PeerID(id))

	if p != nil {
		L.Push(lua.LString(p.username))
	} else {
		L.Push(lua.LNil)
	}

	return 1
}

func luaGetPeerID(L *lua.LState) int {
	name := L.ToString(1)
	l := GetListener()

	found := false
	i := PeerIDCltMin
	for l.id2peer[i].Peer != nil {
		if string(l.id2peer[i].username) == name {
			found = true
			L.Push(lua.LNumber(i))
			break
		}

		i++
	}

	if !found {
		L.Push(lua.LNil)
	}

	return 1
}

func kickPlayer(L *lua.LState) int {
	id := L.ToInt(1)
	reason := L.ToString(2)
	l := GetListener()
	p := l.GetPeerByID(PeerID(id))

	if reason == "" {
		reason = "Kicked."
	} else {
		reason = "Kicked. " + reason
	}

	msg := []byte(reason)

	data := make([]byte, 6+len(msg))
	data[0] = uint8(0x00)
	data[1] = uint8(0x0A)
	data[2] = uint8(0x0A)
	binary.BigEndian.PutUint16(data[3:5], uint16(len(msg)))
	copy(data[5:5+len(msg)], msg)
	data[5+len(msg)] = uint8(0x00)

	ack, err := p.Send(Pkt{Data: data})
	if err != nil {
		log.Print(err)
	}
	<-ack

	p.SendDisco(0, true)
	p.Close()

	return 0
}

func getCurrentServer(L *lua.LState) int {
	id := L.ToInt(1)
	l := GetListener()
	p := l.GetPeerByID(PeerID(id))

	servers := GetConfKey("servers").(map[interface{}]interface{})
	for server := range servers {
		if GetConfKey("servers:"+server.(string)+":address") == p.Server().Addr().String() {
			L.Push(lua.LString(server.(string)))
			break
		}
	}

	return 1
}

func getPlayerAddress(L *lua.LState) int {
	id := L.ToInt(1)
	l := GetListener()
	p := l.GetPeerByID(PeerID(id))

	if p != nil {
		L.Push(lua.LString(p.Addr().String()))
	} else {
		L.Push(lua.LNil)
	}

	return 1
}

func getConnectedPlayers(L *lua.LState) int {
	l := GetListener()

	r := L.NewTable()
	i := PeerIDCltMin
	for l.id2peer[i].Peer != nil {
		r.Append(lua.LNumber(i))
		i++
	}

	L.Push(r)

	return 1
}
