package multiserver

import (
	"encoding/binary"
	"log"
	
	"github.com/yuin/gopher-lua"
)

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
	
	data := make([]byte, 6 + len(msg))
	data[0] = uint8(0x00)
	data[1] = uint8(0x0A)
	data[2] = uint8(0x0A)
	binary.BigEndian.PutUint16(data[3:5], uint16(len(msg)))
	copy(data[5:5 + len(msg)], msg)
	data[5 + len(msg)] = uint8(0x00)
	
	ack, err := p.Send(Pkt{Data: data, ChNo: 0, Unrel: false})
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
		if GetConfKey("servers:" + server.(string) + ":address") == p.Server().Addr().String() {
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
