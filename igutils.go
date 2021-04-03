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

func SendChatMsg(c *Conn, msg string) {
	if c != nil {
		c.SendChatMsg(msg)
	} else {
		log.Print(msg)
	}
}

func init() {
	disable, ok := ConfKey("disable_builtin").(bool)
	if ok && disable {
		return
	}

	RegisterChatCommand("help",
		"Shows the help for a command. Shows the help for all commands if executed without arguments. Usage: help [command]",
		nil,
		true,
		func(c *Conn, param string) {
			showHelp := func(name string) {
				cmd := chatCommands[name]
				if help := cmd.Help(); help != "" {
					if c != nil {
						color := "#F00"
						if has, err := c.CheckPrivs(cmd.privs); (err == nil && has) || cmd.privs == nil {
							color = "#0F0"
						}

						c.SendChatMsg(Colorize(name, color) + ": " + help)
					} else {
						parts := strings.Split(help, "\n")
						for _, part := range parts {
							log.Print(name + ": " + part)
						}
					}
				} else {
					SendChatMsg(c, "No help available for "+name+".")
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
		"Sends a player to a server. Usage: send <playername> <servername>",
		privs("send"),
		true,
		func(c *Conn, param string) {
			if param == "" {
				SendChatMsg(c, "Usage: send <playername> <servername>")
				return
			}

			name := strings.Split(param, " ")[0]
			if name == "" || len(strings.Split(param, " ")) < 2 {
				SendChatMsg(c, "Usage: send <playername> <servername>")
				return
			}
			tosrv := strings.Split(param, " ")[1]
			if tosrv == "" {
				SendChatMsg(c, "Usage: send <playername> <servername>")
				return
			}

			servers := ConfKey("servers").(map[interface{}]interface{})
			if servers[tosrv] == nil {
				SendChatMsg(c, "Unknown servername "+tosrv)
				return
			}

			c2 := ConnByUsername(name)
			if c2 == nil {
				SendChatMsg(c, name+" is not online.")
				return
			}

			srv := c2.ServerName()
			if srv == tosrv {
				SendChatMsg(c, name+" is already connected to this server!")
				return
			}

			go c2.Redirect(tosrv)
		})

	RegisterChatCommand("sendcurrent",
		"Sends all players on the current server to a new server. Usage: sendcurrent <servername>",
		privs("send"),
		false,
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
						go c.Redirect(param)
					}
				}
			}()
		})

	RegisterChatCommand("sendall",
		"Sends all players to a server. Usage: sendall <servername>",
		privs("send"),
		true,
		func(c *Conn, param string) {
			if param == "" {
				SendChatMsg(c, "Usage: sendall <servername>")
				return
			}

			servers := ConfKey("servers").(map[interface{}]interface{})
			if servers[param] == nil {
				SendChatMsg(c, "Unknown servername "+param)
				return
			}

			go func() {
				for _, c := range Conns() {
					if psrv := c.ServerName(); psrv != param {
						go c.Redirect(param)
					}
				}
			}()
		})

	RegisterChatCommand("alert",
		"Sends a message to all players that are connected to the network. Usage: alert [message]",
		privs("alert"),
		true,
		func(c *Conn, param string) {
			ChatSendAll("[ALERT] " + param)
		})

	RegisterChatCommand("server",
		`Prints your current server and a list of all servers if executed without arguments. 
		Sends you to a server if executed with arguments and the required privilege. Usage: server [servername]"`,
		nil,
		false,
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
		"Prints the online status and the current server of a player. Usage: find <playername>",
		privs("find"),
		true,
		func(c *Conn, param string) {
			if param == "" {
				SendChatMsg(c, "Usage: find <playername>")
				return
			}

			c2 := ConnByUsername(param)
			if c2 == nil {
				SendChatMsg(c, param+" is not online.")
			} else {
				srv := c2.ServerName()
				SendChatMsg(c, param+" is connected to server "+srv+".")
			}
		})

	RegisterChatCommand("addr",
		"Prints the network address (including the port) of a connected player. Usage: addr <playername>",
		privs("addr"),
		true,
		func(c *Conn, param string) {
			if param == "" {
				SendChatMsg(c, "Usage: addr <playername>")
				return
			}

			c2 := ConnByUsername(param)
			if c2 == nil {
				SendChatMsg(c, param+" is not online.")
			} else {
				SendChatMsg(c, param+"'s address is "+c2.Addr().String())
			}
		})

	RegisterChatCommand("end",
		"Kicks all connected clients and stops the proxy. Usage: end",
		privs("end"),
		true,
		func(c *Conn, param string) {
			End(false, false)
		})

	RegisterChatCommand("privs",
		`Prints your privileges if executed without arguments. 
		Prints a connected player's privileges if executed with arguments. Usage: privs [playername]`,
		nil,
		true,
		func(c *Conn, param string) {
			var r string

			name := param
			if name == "" {
				if c == nil {
					log.Print("Cannot read privileges of console!")
					return
				}

				name = c.Username()
				r += "Your privileges: "
			} else {
				r += name + "'s privileges: "
			}

			privs, err := Privs(name)
			if err != nil {
				log.Print(err)
				SendChatMsg(c, "An internal error occured while attempting to get the privileges.")
				return
			}

			eprivs := encodePrivs(privs)

			SendChatMsg(c, r+strings.Replace(eprivs, "|", " ", -1))
		})

	RegisterChatCommand("grant",
		`Grants privileges to a connected player. The privileges need to be comma-seperated. 
		If the playername is omitted, privileges are granted to you. Usage: grant [playername] <privileges>`,
		privs("privs"),
		true,
		func(c *Conn, param string) {
			name := strings.Split(param, " ")[0]
			var privnames string

			if len(strings.Split(param, " ")) < 2 {
				if c == nil {
					log.Print("Cannot write privileges of console!")
					return
				}

				privnames = name
				name = c.Username()
			} else {
				privnames = strings.Split(param, " ")[1]
			}

			privs, err := Privs(name)
			if err != nil {
				log.Print(err)
				SendChatMsg(c, "An internal error occured while attempting to get the privileges.")
				return
			}

			splitprivs := strings.Split(strings.Replace(privnames, " ", "", -1), ",")
			for i := range splitprivs {
				privs[splitprivs[i]] = true
			}

			err = SetPrivs(name, privs)
			if err != nil {
				log.Print(err)
				SendChatMsg(c, "An internal error occured while attempting to set the privileges.")
				return
			}

			SendChatMsg(c, "Privileges updated.")
		})

	RegisterChatCommand("revoke",
		`Revokes privileges from a connected player. The privileges need to be comma-seperated. 
		If the playername is omitted, privileges are revoked from you. Usage: revoke [playername] <privileges>`,
		privs("privs"),
		true,
		func(c *Conn, param string) {
			name := strings.Split(param, " ")[0]
			var privnames string

			if len(strings.Split(param, " ")) < 2 {
				if c == nil {
					log.Print("Cannot write privileges of console!")
					return
				}

				privnames = name
				name = c.Username()
			} else {
				privnames = strings.Split(param, " ")[1]
			}

			privs, err := Privs(name)
			if err != nil {
				log.Print(err)
				SendChatMsg(c, "An internal error occured while attempting to get the privileges.")
				return
			}

			splitprivs := strings.Split(strings.Replace(privnames, " ", "", -1), ",")
			for i := range splitprivs {
				privs[splitprivs[i]] = false
			}

			err = SetPrivs(name, privs)
			if err != nil {
				log.Print(err)
				SendChatMsg(c, "An internal error occured while attempting to set the privileges.")
				return
			}

			SendChatMsg(c, "Privileges updated.")
		})

	RegisterChatCommand("banlist",
		"Prints the list of banned IP address and associated players. Usage: banlist",
		privs("ban"),
		true,
		func(c *Conn, param string) {
			bans, err := BanList()
			if err != nil {
				SendChatMsg(c, "An internal error occured while attempting to read the ban list.")
				return
			}

			msg := "Address | Name\n"
			for addr, name := range bans {
				msg += addr + " | " + name + "\n"
			}

			SendChatMsg(c, msg)
		})

	RegisterChatCommand("ban",
		"Bans an IP address or a connected player. Usage: ban <playername | IP address>",
		privs("ban"),
		true,
		func(c *Conn, param string) {
			if param == "" {
				SendChatMsg(c, "Usage: ban <playername | IP address>")
				return
			}

			err := Ban(param)
			if err != nil {
				c2 := ConnByUsername(param)
				if c2 == nil {
					SendChatMsg(c, param+" is not online.")
					return
				}

				if err := c2.Ban(); err != nil {
					SendChatMsg(c, "An internal error occured while attempting to ban the player.")
					return
				}
			}

			SendChatMsg(c, "Banned "+param)
		})

	RegisterChatCommand("unban",
		"Unbans an IP address or a playername. Usage: unban <playername | IP address>",
		privs("ban"),
		true,
		func(c *Conn, param string) {
			if param == "" {
				SendChatMsg(c, "Usage: unban <playername | IP address>")
				return
			}

			if err := Unban(param); err != nil {
				SendChatMsg(c, "An internal error occured while attempting to unban the player.")
				return
			}

			SendChatMsg(c, "Unbanned "+param)
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

	go func() {
		<-LogReady()
		log.Print("Loaded builtin")
	}()
}
