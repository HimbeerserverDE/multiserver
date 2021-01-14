package main

import (
	"github.com/HimbeerserverDE/multiserver"
	"strings"
)

func cmdSend(p *multiserver.Peer, param string) {
	if param == "" {
		p.SendChatMsg("Usage: #send <playername> <servername>")
		return
	}

	name := strings.Split(param, " ")[0]
	if name == "" || len(strings.Split(param, " ")) < 2 {
		p.SendChatMsg("Usage: #send <playername> <servername>")
		return
	}
	tosrv := strings.Split(param, " ")[1]
	if tosrv == "" {
		p.SendChatMsg("Usage: #send <playername> <servername>")
		return
	}

	servers := multiserver.GetConfKey("servers").(map[interface{}]interface{})
	if servers[tosrv] == nil {
		p.SendChatMsg("Unknown servername " + tosrv)
		return
	}

	p2 := multiserver.GetListener().GetPeerByName(name)
	if p2 == nil {
		p.SendChatMsg(name + " is not online.")
		return
	}

	var srv string
	for server := range servers {
		if multiserver.GetConfKey("servers:"+server.(string)+":address") == p2.Server().Addr().String() {
			srv = server.(string)
			break
		}
	}

	if srv == tosrv {
		p.SendChatMsg(name + " is already connected to this server!")
	}

	p2.Redirect(tosrv)
}

func init() {
	privs := make(map[string]map[string]bool)

	privs["send"] = make(map[string]bool)
	privs["send"]["send"] = true

	privs["sendcurrent"] = make(map[string]bool)
	privs["sendcurrent"]["send"] = true

	privs["sendall"] = make(map[string]bool)
	privs["sendall"]["send"] = true

	multiserver.RegisterChatCommand("send", privs["send"], cmdSend)

	multiserver.RegisterChatCommand("sendcurrent", privs["sendcurrent"],
		func(p *multiserver.Peer, param string) {
			if param == "" {
				p.SendChatMsg("Usage: #sendcurrent <servername>")
				return
			}

			servers := multiserver.GetConfKey("servers").(map[interface{}]interface{})
			if servers[param] == nil {
				p.SendChatMsg("Unknown servername " + param)
			}

			var srv string
			for server := range servers {
				if multiserver.GetConfKey("servers:"+server.(string)+":address") == p.Server().Addr().String() {
					srv = server.(string)
					break
				}
			}

			if srv == param {
				p.SendChatMsg("All targets are already connected to this server!")
				return
			}

			p.Redirect(param)
			peers := multiserver.GetListener().GetPeers()
			for i := range peers {
				var psrv string
				for server := range servers {
					if multiserver.GetConfKey("servers:"+server.(string)+":address") == peers[i].Server().Addr().String() {
						psrv = server.(string)
						break
					}
				}

				if psrv == srv {
					peers[i].Redirect(param)
				}
			}
		})

	multiserver.RegisterChatCommand("sendall", privs["sendall"],
		func(p *multiserver.Peer, param string) {
			if param == "" {
				p.SendChatMsg("Usage: #sendall <servername>")
				return
			}

			servers := multiserver.GetConfKey("servers").(map[interface{}]interface{})
			if servers[param] == nil {
				p.SendChatMsg("Unknown servername " + param)
			}

			var srv string
			for server := range servers {
				if multiserver.GetConfKey("servers:"+server.(string)+":address") == p.Server().Addr().String() {
					srv = server.(string)
					break
				}
			}

			if srv != param {
				p.Redirect(param)
			}
			peers := multiserver.GetListener().GetPeers()
			for i := range peers {
				var psrv string
				for server := range servers {
					if multiserver.GetConfKey("servers:"+server.(string)+":address") == peers[i].Server().Addr().String() {
						psrv = server.(string)
						break
					}
				}

				if psrv != param {
					peers[i].Redirect(param)
				}
			}
		})
}
