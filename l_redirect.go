package multiserver

import (
	"github.com/yuin/gopher-lua"
)

func redirect(L *lua.LState) int {
	id := PeerID(L.ToInt(1))
	srv := L.ToString(2)
	l := GetListener()
	p := l.GetPeerByID(id)
	
	p.Redirect(srv)
	
	return 0
}
