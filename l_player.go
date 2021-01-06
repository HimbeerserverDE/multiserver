package multiserver

import (
	"github.com/yuin/gopher-lua"
)

func getPlayerName(L *lua.LState) int {
	id := L.ToInt(1)
	l := GetListener()
	p := l.GetPeerByID(PeerID(id))
	
	L.Push(lua.LString(p.username))
	
	return 1
}
