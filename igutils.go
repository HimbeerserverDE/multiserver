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
	disable, ok := ConfKey("disable_builtin").(bool)
	if ok && disable {
		return
	}

	RegisterChatCommand("help",
		nil,
		"Shows the help for a command. Shows the help for all commands if executed without arguments. Usage: help [command]",
		func(p *Peer, param string) {
			showHelp := func(name string) {
				cmd := chatCommands[name]
				if help := cmd.Help(); help != "" {
					color := "#F00"
					if has, err := p.CheckPrivs(cmd.privs); (err == nil && has) || cmd.privs == nil {
						color = "#0F0"
					}

					p.SendChatMsg(Colorize(name, color) + ": " + help)
				} else {
					p.SendChatMsg("No help available for " + name + ".")
				}
			}

			if param == "" {
				for cmd := range chatCommands {
					showHelp(cmd)
				}
			} else {
				showHelp(param)
			}
		})

	RegisterChatCommand("send",
		privs("send"),
		"Sends a player to a server. Usage: send <playername> <servername>",
		func(p *Peer, param string) {
			if param == "" {
				p.SendChatMsg("Usage: send <playername> <servername>")
				return
			}

			name := strings.Split(param, " ")[0]
			if name == "" || len(strings.Split(param, " ")) < 2 {
				p.SendChatMsg("Usage: send <playername> <servername>")
				return
			}
			tosrv := strings.Split(param, " ")[1]
			if tosrv == "" {
				p.SendChatMsg("Usage: send <playername> <servername>")
				return
			}

			servers := ConfKey("servers").(map[interface{}]interface{})
			if servers[tosrv] == nil {
				p.SendChatMsg("Unknown servername " + tosrv)
				return
			}

			p2 := PeerByUsername(name)
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

	RegisterChatCommand("sendcurrent",
		privs("send"),
		"Sends all players on the current server to a new server. Usage: sendcurrent <servername>",
		func(p *Peer, param string) {
			if param == "" {
				p.SendChatMsg("Usage: sendcurrent <servername>")
				return
			}

			servers := ConfKey("servers").(map[interface{}]interface{})
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
				for _, p := range Peers() {
					if p.ServerName() == srv {
						p.Redirect(param)
					}
				}
			}()
		})

	RegisterChatCommand("sendall",
		privs("send"),
		"Sends all players to a server. Usage: sendall <servername>",
		func(p *Peer, param string) {
			if param == "" {
				p.SendChatMsg("Usage: sendall <servername>")
				return
			}

			servers := ConfKey("servers").(map[interface{}]interface{})
			if servers[param] == nil {
				p.SendChatMsg("Unknown servername " + param)
				return
			}

			go func() {
				for _, p := range Peers() {
					if psrv := p.ServerName(); psrv != param {
						p.Redirect(param)
					}
				}
			}()
		})

	RegisterChatCommand("alert",
		privs("alert"),
		"Sends a message to all players that are connected to the network. Usage: alert [message]",
		func(p *Peer, param string) {
			ChatSendAll("[ALERT] " + param)
		})

	RegisterChatCommand("server",
		nil,
		`Prints your current server and a list of all servers if executed without arguments. 
		Sends you to a server if executed with arguments and the required privilege. Usage: server [servername]"`,
		func(p *Peer, param string) {
			if param == "" {
				var r string
				servers := ConfKey("servers").(map[interface{}]interface{})
				for server := range servers {
					r += server.(string) + " "
				}
				srv := p.ServerName()
				p.SendChatMsg("Current server: " + srv + " | All servers: " + r)
			} else {
				servers := ConfKey("servers").(map[interface{}]interface{})
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
				reqpriv, ok := ConfKey("servers:" + param + ":priv").(string)
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

	RegisterChatCommand("find",
		privs("find"),
		"Prints the online status and the current server of a player. Usage: find <playername>",
		func(p *Peer, param string) {
			if param == "" {
				p.SendChatMsg("Usage: find <playername>")
				return
			}

			p2 := PeerByUsername(param)
			if p2 == nil {
				p.SendChatMsg(param + " is not online.")
			} else {
				srv := p2.ServerName()
				p.SendChatMsg(param + " is connected to server " + srv + ".")
			}
		})

	RegisterChatCommand("addr",
		privs("addr"),
		"Prints the network address (including the port) of a connected player. Usage: addr <playername>",
		func(p *Peer, param string) {
			if param == "" {
				p.SendChatMsg("Usage: addr <playername>")
				return
			}

			p2 := PeerByUsername(param)
			if p2 == nil {
				p.SendChatMsg(param + " is not online.")
			} else {
				p.SendChatMsg(param + "'s address is " + p2.Addr().String())
			}
		})

	RegisterChatCommand("end",
		privs("end"),
		"Kicks all connected clients and stops the proxy. Usage: end",
		func(p *Peer, param string) {
			go End(false, false)
		})

	RegisterChatCommand("privs",
		nil,
		`Prints your privileges if executed without arguments. 
		Prints a connected player's privileges if executed with arguments. Usage: privs [playername]`,
		func(p *Peer, param string) {
			var r string

			name := param
			var p2 *Peer
			if name == "" {
				p2 = p
				r += "Your privileges: "
			} else {
				p2 = PeerByUsername(name)
				r += name + "'s privileges: "
			}

			if name != "" && !IsOnline(name) {
				p.SendChatMsg(name + " is not online.")
				return
			}

			privs, err := p2.Privs()
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

	RegisterChatCommand("grant",
		privs("privs"),
		`Grants privileges to a connected player. The privileges need to be comma-seperated. 
		If the playername is omitted, privileges are granted to you. Usage: grant [playername] <privileges>`,
		func(p *Peer, param string) {
			name := strings.Split(param, " ")[0]
			var privnames string
			var p2 *Peer
			if len(strings.Split(param, " ")) < 2 {
				p2 = p
				privnames = name
			} else {
				p2 = PeerByUsername(name)
				privnames = strings.Split(param, " ")[1]
			}

			if len(strings.Split(param, " ")) >= 2 && !IsOnline(name) {
				p.SendChatMsg(name + " is not online.")
				return
			}

			privs, err := p2.Privs()
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

	RegisterChatCommand("revoke",
		privs("privs"),
		`Revokes privileges from a connected player. The privileges need to be comma-seperated. 
		If the playername is omitted, privileges are revoked from you. Usage: revoke [playername] <privileges>`,
		func(p *Peer, param string) {
			name := strings.Split(param, " ")[0]
			var privnames string
			var p2 *Peer
			if len(strings.Split(param, " ")) < 2 {
				p2 = p
				privnames = name
			} else {
				p2 = PeerByUsername(name)
				privnames = strings.Split(param, " ")[1]
			}

			if len(strings.Split(param, " ")) >= 2 && !IsOnline(name) {
				p.SendChatMsg(name + " is not online.")
				return
			}

			privs, err := p2.Privs()
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

	RegisterChatCommand("banlist",
		privs("ban"),
		"Prints the list of banned IP address and associated players. Usage: banlist",
		func(p *Peer, param string) {
			bans, err := BanList()
			if err != nil {
				p.SendChatMsg("An internal error occured while attempting to read the ban list.")
				return
			}

			msg := "Address | Name\n"
			for addr, name := range bans {
				msg += addr + " | " + name + "\n"
			}

			p.SendChatMsg(msg)
		})

	RegisterChatCommand("ban",
		privs("ban"),
		"Bans an IP address or a connected player. Usage: ban <playername | IP address>",
		func(p *Peer, param string) {
			if param == "" {
				p.SendChatMsg("Usage: ban <playername | IP address>")
				return
			}

			err := Ban(param)
			if err != nil {
				p2 := PeerByUsername(param)
				if p2 == nil {
					p.SendChatMsg(param + " is not online.")
					return
				}

				if err := p2.Ban(); err != nil {
					p.SendChatMsg("An internal error occured while attempting to ban the player.")
					return
				}
			}

			p.SendChatMsg("Banned " + param)
		})

	RegisterChatCommand("unban",
		privs("ban"),
		"Unbans an IP address or a playername. Usage: unban <playername | IP address>",
		func(p *Peer, param string) {
			if param == "" {
				p.SendChatMsg("Usage: unban <playername | IP address>")
				return
			}

			if err := Unban(param); err != nil {
				p.SendChatMsg("An internal error occured while attempting to unban the player.")
				return
			}

			p.SendChatMsg("Unbanned " + param)
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
