package multiserver

import (
	"log"
	
	"github.com/yuin/gopher-lua"
)

var redirectDoneHandlers []*lua.LFunction

func registerOnRedirectDone(L *lua.LState) int {
	f := L.ToFunction(1)
	redirectDoneHandlers = append(redirectDoneHandlers, f)
	
	return 0
}

func processRedirectDone(p *Peer, newsrv string) {
	var srv string
	
	servers := GetConfKey("servers").(map[interface{}]interface{})
	for server := range servers {
		if GetConfKey("servers:" + server.(string) + ":address") == p.Server().Addr().String() {
			srv = server.(string)
			
			break
		}
	}
	
	success := srv == newsrv
	
	for i := range redirectDoneHandlers {
		if err := l.CallByParam(lua.P{Fn: redirectDoneHandlers[i], NRet: 0, Protect: true}, lua.LNumber(p.ID()), lua.LString(newsrv), lua.LBool(success)); err != nil {
			log.Print(err)
			
			End(true, true)
		}
	}
}

func redirect(L *lua.LState) int {
	id := PeerID(L.ToInt(1))
	srv := L.ToString(2)
	l := GetListener()
	p := l.GetPeerByID(id)
	
	go p.Redirect(srv)
	
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
