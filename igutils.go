package main

import (
	"log"
	"strings"
)

func privs(args ...string) map[string]bool {
	m := make(map[string]bool)
	for _, priv := range args {
		m[priv] = true
	}
	return m
}

func init() {
	disable, ok := GetConfKey("disable_builtin").(bool)
	if ok && disable {
		return
	}

	RegisterChatCommand("send", privs("send"),
		func(p *Peer, param string) {
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

			servers := GetConfKey("servers").(map[interface{}]interface{})
			if servers[tosrv] == nil {
				p.SendChatMsg("Unknown servername " + tosrv)
				return
			}

			p2 := GetListener().GetPeerByUsername(name)
			if p2 == nil {
				p.SendChatMsg(name + " is not online.")
				return
			}

			srv := p2.ServerName()
			if srv == tosrv {
				p.SendChatMsg(name + " is already connected to this server!")
			}

			go p2.Redirect(tosrv)
		})

	RegisterChatCommand("sendcurrent", privs("send"),
		func(p *Peer, param string) {
			if param == "" {
				p.SendChatMsg("Usage: #sendcurrent <servername>")
				return
			}

			servers := GetConfKey("servers").(map[interface{}]interface{})
			if servers[param] == nil {
				p.SendChatMsg("Unknown servername " + param)
				return
			}

			srv := p.ServerName()
			if srv == param {
				p.SendChatMsg("All targets are already connected to this server!")
				return
			}

			go func() {
				peers := GetListener().GetPeers()
				for i := range peers {
					if peers[i].ServerName() == srv {
						peers[i].Redirect(param)
					}
				}
			}()
		})

	RegisterChatCommand("sendall", privs("send"),
		func(p *Peer, param string) {
			if param == "" {
				p.SendChatMsg("Usage: #sendall <servername>")
				return
			}

			servers := GetConfKey("servers").(map[interface{}]interface{})
			if servers[param] == nil {
				p.SendChatMsg("Unknown servername " + param)
				return
			}

			go func() {
				peers := GetListener().GetPeers()
				for i := range peers {
					if psrv := peers[i].ServerName(); psrv != param {
						peers[i].Redirect(param)
					}
				}
			}()
		})

	RegisterChatCommand("alert", privs("alert"),
		func(p *Peer, param string) {
			ChatSendAll("[ALERT] " + param)
		})

	RegisterChatCommand("server", nil,
		func(p *Peer, param string) {
			if param == "" {
				var r string
				servers := GetConfKey("servers").(map[interface{}]interface{})
				for server := range servers {
					r += server.(string) + " "
				}
				srv := p.ServerName()
				p.SendChatMsg("Current server: " + srv + " | All servers: " + r)
			} else {
				servers := GetConfKey("servers").(map[interface{}]interface{})
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
				reqpriv, ok := GetConfKey("servers:" + param + ":priv").(string)
				if ok {
					reqprivs[reqpriv] = true
				}

				allow, err := p.CheckPrivs(reqprivs)
				if err != nil {
					log.Print(err)
					p.SendChatMsg("An internal error occured while attempting to check your privileges.")
					return
				}

				if !allow {
					p.SendChatMsg("You do not have permission to join this server! Required privilege: " + reqpriv)
					return
				}

				go p.Redirect(param)
				p.SendChatMsg("Redirecting you to " + param + ".")
			}
		})

	RegisterChatCommand("find", privs("find"),
		func(p *Peer, param string) {
			if param == "" {
				p.SendChatMsg("Usage: #find <playername>")
				return
			}

			p2 := GetListener().GetPeerByUsername(param)
			if p2 == nil {
				p.SendChatMsg(param + " is not online.")
			} else {
				srv := p2.ServerName()
				p.SendChatMsg(param + " is connected to server " + srv + ".")
			}
		})

	RegisterChatCommand("addr", privs("addr"),
		func(p *Peer, param string) {
			if param == "" {
				p.SendChatMsg("Usage: #addr <playername>")
				return
			}

			p2 := GetListener().GetPeerByUsername(param)
			if p2 == nil {
				p.SendChatMsg(param + " is not online.")
			} else {
				p.SendChatMsg(param + "'s address is " + p2.Addr().String())
			}
		})

	RegisterChatCommand("end", privs("end"),
		func(p *Peer, param string) {
			go End(false, false)
		})

	RegisterChatCommand("privs", nil,
		func(p *Peer, param string) {
			var r string

			name := param
			var p2 *Peer
			if name == "" {
				p2 = p
				r += "Your privileges: "
			} else {
				p2 = GetListener().GetPeerByUsername(name)
				r += name + "'s privileges: "
			}

			if name != "" && !IsOnline(name) {
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

	RegisterChatCommand("grant", privs("privs"),
		func(p *Peer, param string) {
			name := strings.Split(param, " ")[0]
			var privnames string
			var p2 *Peer
			if len(strings.Split(param, " ")) < 2 {
				p2 = p
				privnames = name
			} else {
				p2 = GetListener().GetPeerByUsername(name)
				privnames = strings.Split(param, " ")[1]
			}

			if len(strings.Split(param, " ")) >= 2 && !IsOnline(name) {
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

	RegisterChatCommand("revoke", privs("privs"),
		func(p *Peer, param string) {
			name := strings.Split(param, " ")[0]
			var privnames string
			var p2 *Peer
			if len(strings.Split(param, " ")) < 2 {
				p2 = p
				privnames = name
			} else {
				p2 = GetListener().GetPeerByUsername(name)
				privnames = strings.Split(param, " ")[1]
			}

			if len(strings.Split(param, " ")) >= 2 && !IsOnline(name) {
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

	RegisterOnRedirectDone(func(p *Peer, newsrv string, success bool) {
		if success {
			err := SetStorageKey("server:"+p.Username(), newsrv)
			if err != nil {
				log.Print(err)
				return
			}
		} else {
			p.SendChatMsg("Could not connect you to " + newsrv + "!")
		}
	})

	log.Print("Loaded builtin")
}
