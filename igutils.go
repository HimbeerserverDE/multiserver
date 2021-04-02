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
		func(c *Conn, param string) {
			showHelp := func(name string) {
				cmd := chatCommands[name]
				if help := cmd.Help(); help != "" {
					color := "#F00"
					if has, err := c.CheckPrivs(cmd.privs); (err == nil && has) || cmd.privs == nil {
						color = "#0F0"
					}

					c.SendChatMsg(Colorize(name, color) + ": " + help)
				} else {
					c.SendChatMsg("No help available for " + name + ".")
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
		func(c *Conn, param string) {
			if param == "" {
				c.SendChatMsg("Usage: send <playername> <servername>")
				return
			}

			name := strings.Split(param, " ")[0]
			if name == "" || len(strings.Split(param, " ")) < 2 {
				c.SendChatMsg("Usage: send <playername> <servername>")
				return
			}
			tosrv := strings.Split(param, " ")[1]
			if tosrv == "" {
				c.SendChatMsg("Usage: send <playername> <servername>")
				return
			}

			servers := ConfKey("servers").(map[interface{}]interface{})
			if servers[tosrv] == nil {
				c.SendChatMsg("Unknown servername " + tosrv)
				return
			}

			c2 := ConnByUsername(name)
			if c2 == nil {
				c.SendChatMsg(name + " is not online.")
				return
			}

			srv := c2.ServerName()
			if srv == tosrv {
				c.SendChatMsg(name + " is already connected to this server!")
			}

			go c2.Redirect(tosrv)
		})

	RegisterChatCommand("sendcurrent",
		privs("send"),
		"Sends all players on the current server to a new server. Usage: sendcurrent <servername>",
		func(c *Conn, param string) {
			if param == "" {
				c.SendChatMsg("Usage: sendcurrent <servername>")
				return
			}

			servers := ConfKey("servers").(map[interface{}]interface{})
			if servers[param] == nil {
				c.SendChatMsg("Unknown servername " + param)
				return
			}

			srv := c.ServerName()
			if srv == param {
				c.SendChatMsg("All targets are already connected to this server!")
				return
			}

			go func() {
				for _, c := range Conns() {
					if c.ServerName() == srv {
						c.Redirect(param)
					}
				}
			}()
		})

	RegisterChatCommand("sendall",
		privs("send"),
		"Sends all players to a server. Usage: sendall <servername>",
		func(c *Conn, param string) {
			if param == "" {
				c.SendChatMsg("Usage: sendall <servername>")
				return
			}

			servers := ConfKey("servers").(map[interface{}]interface{})
			if servers[param] == nil {
				c.SendChatMsg("Unknown servername " + param)
				return
			}

			go func() {
				for _, c := range Conns() {
					if psrv := c.ServerName(); psrv != param {
						c.Redirect(param)
					}
				}
			}()
		})

	RegisterChatCommand("alert",
		privs("alert"),
		"Sends a message to all players that are connected to the network. Usage: alert [message]",
		func(c *Conn, param string) {
			ChatSendAll("[ALERT] " + param)
		})

	RegisterChatCommand("server",
		nil,
		`Prints your current server and a list of all servers if executed without arguments. 
		Sends you to a server if executed with arguments and the required privilege. Usage: server [servername]"`,
		func(c *Conn, param string) {
			if param == "" {
				var r string
				servers := ConfKey("servers").(map[interface{}]interface{})
				for server := range servers {
					r += server.(string) + " "
				}

				c.SendChatMsg("Current server: " + c.ServerName() + " | All servers: " + r)
			} else {
				servers := ConfKey("servers").(map[interface{}]interface{})
				srv := c.ServerName()

				if srv == param {
					c.SendChatMsg("You are already connected to this server!")
					return
				}

				nogrp := true
				groups, ok := ConfKey("groups").(map[interface{}]interface{})
				if ok {
					for grp := range groups {
						name, ok := grp.(string)
						if !ok {
							continue
						}

						if name == param {
							nogrp = false
						}
					}
				}

				if servers[param] == nil && nogrp {
					c.SendChatMsg("Unknown servername " + param)
					return
				}

				reqprivs := make(map[string]bool)

				reqpriv, ok := ConfKey("servers:" + param + ":priv").(string)
				if ok {
					reqprivs[reqpriv] = true
				}

				reqpriv, ok = ConfKey("group_privs:" + param).(string)
				if ok {
					reqprivs[reqpriv] = true
				}

				allow, err := c.CheckPrivs(reqprivs)
				if err != nil {
					log.Print(err)
					c.SendChatMsg("An internal error occured while attempting to check your privileges.")
					return
				}

				if !allow {
					c.SendChatMsg("You do not have permission to join this server! Required privilege: " + reqpriv)
					return
				}

				go c.Redirect(param)
				c.SendChatMsg("Redirecting you to " + param + ".")
			}
		})

	RegisterChatCommand("find",
		privs("find"),
		"Prints the online status and the current server of a player. Usage: find <playername>",
		func(c *Conn, param string) {
			if param == "" {
				c.SendChatMsg("Usage: find <playername>")
				return
			}

			c2 := ConnByUsername(param)
			if c2 == nil {
				c.SendChatMsg(param + " is not online.")
			} else {
				srv := c2.ServerName()
				c.SendChatMsg(param + " is connected to server " + srv + ".")
			}
		})

	RegisterChatCommand("addr",
		privs("addr"),
		"Prints the network address (including the port) of a connected player. Usage: addr <playername>",
		func(c *Conn, param string) {
			if param == "" {
				c.SendChatMsg("Usage: addr <playername>")
				return
			}

			c2 := ConnByUsername(param)
			if c2 == nil {
				c.SendChatMsg(param + " is not online.")
			} else {
				c.SendChatMsg(param + "'s address is " + c2.Addr().String())
			}
		})

	RegisterChatCommand("end",
		privs("end"),
		"Kicks all connected clients and stops the proxy. Usage: end",
		func(c *Conn, param string) {
			go End(false, false)
		})

	RegisterChatCommand("privs",
		nil,
		`Prints your privileges if executed without arguments. 
		Prints a connected player's privileges if executed with arguments. Usage: privs [playername]`,
		func(c *Conn, param string) {
			var r string

			name := param
			if name == "" {
				name = c.Username()
				r += "Your privileges: "
			} else {
				r += name + "'s privileges: "
			}

			privs, err := Privs(name)
			if err != nil {
				log.Print(err)
				c.SendChatMsg("An internal error occured while attempting to get the privileges.")
				return
			}

			eprivs := encodePrivs(privs)

			c.SendChatMsg(r + strings.Replace(eprivs, "|", " ", -1))
		})

	RegisterChatCommand("grant",
		privs("privs"),
		`Grants privileges to a connected player. The privileges need to be comma-seperated. 
		If the playername is omitted, privileges are granted to you. Usage: grant [playername] <privileges>`,
		func(c *Conn, param string) {
			name := strings.Split(param, " ")[0]
			var privnames string

			if len(strings.Split(param, " ")) < 2 {
				privnames = name
				name = c.Username()
			} else {
				privnames = strings.Split(param, " ")[1]
			}

			privs, err := Privs(name)
			if err != nil {
				log.Print(err)
				c.SendChatMsg("An internal error occured while attempting to get the privileges.")
				return
			}

			splitprivs := strings.Split(strings.Replace(privnames, " ", "", -1), ",")
			for i := range splitprivs {
				privs[splitprivs[i]] = true
			}

			err = SetPrivs(name, privs)
			if err != nil {
				log.Print(err)
				c.SendChatMsg("An internal error occured while attempting to set the privileges.")
				return
			}

			c.SendChatMsg("Privileges updated.")
		})

	RegisterChatCommand("revoke",
		privs("privs"),
		`Revokes privileges from a connected player. The privileges need to be comma-seperated. 
		If the playername is omitted, privileges are revoked from you. Usage: revoke [playername] <privileges>`,
		func(c *Conn, param string) {
			name := strings.Split(param, " ")[0]
			var privnames string

			if len(strings.Split(param, " ")) < 2 {
				privnames = name
				name = c.Username()
			} else {
				privnames = strings.Split(param, " ")[1]
			}

			privs, err := Privs(name)
			if err != nil {
				log.Print(err)
				c.SendChatMsg("An internal error occured while attempting to get the privileges.")
				return
			}

			splitprivs := strings.Split(strings.Replace(privnames, " ", "", -1), ",")
			for i := range splitprivs {
				privs[splitprivs[i]] = false
			}

			err = SetPrivs(name, privs)
			if err != nil {
				log.Print(err)
				c.SendChatMsg("An internal error occured while attempting to set the privileges.")
				return
			}

			c.SendChatMsg("Privileges updated.")
		})

	RegisterChatCommand("banlist",
		privs("ban"),
		"Prints the list of banned IP address and associated players. Usage: banlist",
		func(c *Conn, param string) {
			bans, err := BanList()
			if err != nil {
				c.SendChatMsg("An internal error occured while attempting to read the ban list.")
				return
			}

			msg := "Address | Name\n"
			for addr, name := range bans {
				msg += addr + " | " + name + "\n"
			}

			c.SendChatMsg(msg)
		})

	RegisterChatCommand("ban",
		privs("ban"),
		"Bans an IP address or a connected player. Usage: ban <playername | IP address>",
		func(c *Conn, param string) {
			if param == "" {
				c.SendChatMsg("Usage: ban <playername | IP address>")
				return
			}

			err := Ban(param)
			if err != nil {
				c2 := ConnByUsername(param)
				if c2 == nil {
					c.SendChatMsg(param + " is not online.")
					return
				}

				if err := c2.Ban(); err != nil {
					c.SendChatMsg("An internal error occured while attempting to ban the player.")
					return
				}
			}

			c.SendChatMsg("Banned " + param)
		})

	RegisterChatCommand("unban",
		privs("ban"),
		"Unbans an IP address or a playername. Usage: unban <playername | IP address>",
		func(c *Conn, param string) {
			if param == "" {
				c.SendChatMsg("Usage: unban <playername | IP address>")
				return
			}

			if err := Unban(param); err != nil {
				c.SendChatMsg("An internal error occured while attempting to unban the player.")
				return
			}

			c.SendChatMsg("Unbanned " + param)
		})

	RegisterOnRedirectDone(func(c *Conn, newsrv string, success bool) {
		if success {
			err := SetStorageKey("server:"+c.Username(), newsrv)
			if err != nil {
				log.Print(err)
				return
			}
		} else {
			c.SendChatMsg("Could not connect you to " + newsrv + "!")
		}
	})

	log.Print("Loaded builtin")
}
