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
	privs["addr"]["addr"] = true

	privs["end"] = make(map[string]bool)
	privs["end"]["end"] = true

	privs["grant"] = make(map[string]bool)
	privs["grant"]["privs"] = true

	privs["revoke"] = make(map[string]bool)
	privs["revoke"]["privs"] = true

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
					p.SendChatMsg("An internal error occured while attempting to check your privileges.")
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

	multiserver.RegisterChatCommand("end", privs["end"],
		func(p *multiserver.Peer, param string) {
			go multiserver.End(false, false)
		})

	multiserver.RegisterChatCommand("privs", nil,
		func(p *multiserver.Peer, param string) {
			var r string

			name := param
			var p2 *multiserver.Peer
			if name == "" {
				p2 = p
				r += "Your privileges: "
			} else {
				p2 = multiserver.GetListener().GetPeerByName(name)
				r += name + "'s privileges: "
			}

			if name != "" && !multiserver.IsOnline(name) {
				p.SendChatMsg(name + " is not online.")
				return
			}

			privs, err := p2.GetPrivs()
			if err != nil {
				log.Print(err)
				p.SendChatMsg("An internal error occured while attempting to get the privileges.")
				return
			}

			var privnames []string
			for k, v := range privs {
				if v {
					privnames = append(privnames, k)
				}
			}

			p.SendChatMsg(r + strings.Join(privnames, " "))
		})

	multiserver.RegisterChatCommand("grant", privs["grant"],
		func(p *multiserver.Peer, param string) {
			name := strings.Split(param, " ")[0]
			var privnames string
			var p2 *multiserver.Peer
			if len(strings.Split(param, " ")) < 2 {
				p2 = p
				privnames = name
			} else {
				p2 = multiserver.GetListener().GetPeerByName(name)
				privnames = strings.Split(param, " ")[1]
			}

			if len(strings.Split(param, " ")) >= 2 && !multiserver.IsOnline(name) {
				p.SendChatMsg(name + " is not online.")
				return
			}

			privs, err := p2.GetPrivs()
			if err != nil {
				log.Print(err)
				p.SendChatMsg("An internal error occured while attempting to get the privileges.")
				return
			}
			splitprivs := strings.Split(strings.Replace(privnames, " ", "", -1), ",")
			for i := range splitprivs {
				privs[splitprivs[i]] = true
			}
			err = p2.SetPrivs(privs)
			if err != nil {
				log.Print(err)
				p.SendChatMsg("An internal error occured while attempting to get the privileges.")
				return
			}

			p.SendChatMsg("Privileges updated.")
		})

	multiserver.RegisterChatCommand("revoke", privs["revoke"],
		func(p *multiserver.Peer, param string) {
			name := strings.Split(param, " ")[0]
			var privnames string
			var p2 *multiserver.Peer
			if len(strings.Split(param, " ")) < 2 {
				p2 = p
				privnames = name
			} else {
				p2 = multiserver.GetListener().GetPeerByName(name)
				privnames = strings.Split(param, " ")[1]
			}

			if len(strings.Split(param, " ")) >= 2 && !multiserver.IsOnline(name) {
				p.SendChatMsg(name + " is not online.")
				return
			}

			privs, err := p2.GetPrivs()
			if err != nil {
				log.Print(err)
				p.SendChatMsg("An internal error occured while attempting to get the privileges.")
				return
			}
			splitprivs := strings.Split(strings.Replace(privnames, " ", "", -1), ",")
			for i := range splitprivs {
				privs[splitprivs[i]] = false
			}
			err = p2.SetPrivs(privs)
			if err != nil {
				log.Print(err)
				p.SendChatMsg("An internal error occured while attempting to set the privileges.")
				return
			}

			p.SendChatMsg("Privileges updated.")
		})
}
