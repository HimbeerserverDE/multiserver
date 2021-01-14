package main

import (
	"fmt"
	"github.com/HimbeerserverDE/multiserver"
	"log"
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

	srv := p2.ServerName()
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

	privs["alert"] = make(map[string]bool)
	privs["alert"]["alert"] = true

	privs["find"] = make(map[string]bool)
	privs["find"]["find"] = true

	privs["addr"] = make(map[string]bool)
	privs["addr"]["addr"] = false

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

			srv := p.ServerName()
			if srv == param {
				p.SendChatMsg("All targets are already connected to this server!")
				return
			}

			go p.Redirect(param)
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
					go peers[i].Redirect(param)
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

			srv := p.ServerName()
			if srv != param {
				go p.Redirect(param)
			}
			peers := multiserver.GetListener().GetPeers()
			for i := range peers {
				if psrv := peers[i].ServerName(); psrv != param {
					go peers[i].Redirect(param)
				}
			}
		})

	multiserver.RegisterChatCommand("alert", privs["alert"],
		func(p *multiserver.Peer, param string) {
			multiserver.ChatSendAll("[ALERT] " + param)
		})

	multiserver.RegisterChatCommand("server", nil,
		func(p *multiserver.Peer, param string) {
			if param == "" {
				var r string
				servers := multiserver.GetConfKey("servers").(map[interface{}]interface{})
				for server := range servers {
					r += server.(string) + " "
				}
				srv := p.ServerName()
				p.SendChatMsg("Current server: " + srv + " | All servers: " + r)
			} else {
				servers := multiserver.GetConfKey("servers").(map[interface{}]interface{})
				srv := p.ServerName()

				if srv == param {
					p.SendChatMsg("You are already connected to this server!")
					return
				}

				if servers[param] == nil {
					p.SendChatMsg("Unknown servername " + param)
					return
				}

				reqprivs := make(map[string]bool)
				reqpriv := multiserver.GetConfKey("servers:" + param + ":priv")
				if reqpriv != nil && fmt.Sprintf("%T", reqpriv) == "string" {
					reqprivs[reqpriv.(string)] = true
				}

				allow, err := p.CheckPrivs(reqprivs)
				if err != nil {
					log.Print(err)
					p.SendChatMsg("An internal error occured while trying to check your privileges.")
					return
				}

				if !allow {
					p.SendChatMsg("You do not have permission to join this server! Required privilege: " + reqpriv.(string))
					return
				}

				go p.Redirect(param)
				p.SendChatMsg("Redirecting you to " + param + ".")
			}
		})

	multiserver.RegisterChatCommand("find", privs["find"],
		func(p *multiserver.Peer, param string) {
			if param == "" {
				p.SendChatMsg("Usage: #find <playername>")
				return
			}

			p2 := multiserver.GetListener().GetPeerByName(param)
			if p2 == nil {
				p.SendChatMsg(param + " is not online.")
			} else {
				srv := p2.ServerName()
				p.SendChatMsg(param + " is connected to server " + srv + ".")
			}
		})

	multiserver.RegisterChatCommand("addr", privs["addr"],
		func(p *multiserver.Peer, param string) {
			if param == "" {
				p.SendChatMsg("Usage: #ip <playername>")
				return
			}

			p2 := multiserver.GetListener().GetPeerByName(param)
			if p2 == nil {
				p.SendChatMsg(param + " is not online.")
			} else {
				p.SendChatMsg(param + "'s address is " + p2.Addr().String())
			}
		})
}
