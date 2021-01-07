package multiserver

import "github.com/yuin/gopher-lua"

func redirect(L *lua.LState) int {
	id := PeerID(L.ToInt(1))
	srv := L.ToString(2)
	l := GetListener()
	p := l.GetPeerByID(id)
	
	p.Redirect(srv)
	
	return 0
}

func getServers(L *lua.LState) int {
	servers := GetConfKey("servers").(map[interface{}]interface{})
	
	r := L.NewTable()
	for server := range servers {
		addr := GetConfKey("servers:" + server.(string) + ":address")
		r.RawSet(lua.LString(server.(string)), lua.LString(addr.(string)))
	}
	
	L.Push(r)
	
	return 1
}
