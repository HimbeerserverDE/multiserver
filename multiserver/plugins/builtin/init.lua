multiserver.register_chatcommand("send", {
	privs = {send = true},
	func = function(id, param)
		local name = multiserver.split(param, " ")[1]
		local tosrv = multiserver.split(param, " ")[2]
		
		if not name or name == "" or not tosrv or tosrv == "" then
			return "Usage: /send <playername> <servername>"
		end
		
		if not multiserver.get_servers()[tosrv] then
			return "Unknown servername " .. tosrv
		end
		
		local peerid = multiserver.get_peer_id(name)
		if peerid then
			if multiserver.get_current_server(peerid) == tosrv then
				return name .. " is already connected to this server!"
			end
			
			multiserver.redirect(peerid, tosrv)
			return "Redirecting " .. name .. " to " .. tosrv .. "."
		end
	end,
})

multiserver.register_chatcommand("sendcurrent", {
	privs = {send = true},
	func = function(id, param)
		if not param or param == "" then
			return "Usage: /sendcurrent <servername>"
		end
		
		if not multiserver.get_servers()[param] then
			return "Unkown servername " .. param
		end
		
		if param == multiserver.get_current_server(id) then
			return "All targets are already connected to this server!"
		end
		
		local fromsrv = multiserver.get_current_server(id)
		
		for _, peerid in ipairs(multiserver.get_connected_players()) do
			if multiserver.get_current_server(peerid) == fromsrv then
				multiserver.redirect(peerid, param)
			end
		end
	end,
})

multiserver.register_chatcommand("sendall", {
	privs = {send = true},
	func = function(id, param)
		if not param or param == "" then
			return "Usage: /sendall <servername>"
		end
		
		if not multiserver.get_servers()[param] then
			return "Unkown servername " .. param
		end
		
		for _, peerid in ipairs(multiserver.get_connected_players()) do
			multiserver.log(peerid)
			multiserver.log(multiserver.get_current_server(peerid))
			multiserver.log(param)
			if multiserver.get_current_server(peerid) ~= param then
				multiserver.redirect(peerid, param)
			end
		end
	end,
})

multiserver.register_chatcommand("alert", {
	privs = {alert = true},
	func = function(id, param)
		multiserver.chat_send_all("[ALERT] " .. param)
	end,
})

multiserver.register_chatcommand("server", {
	privs = {},
	func = function(id, param)
		if not param or param == "" then
			local r = ""
			for server, addr in pairs(multiserver.get_servers()) do
				r = r .. server .. " "
			end
			return "Current server: " .. multiserver.get_current_server(id) .. " | All servers: " .. r
		else
			if multiserver.get_current_server(id) == param then
				return "You are already connected to this server!"
			end
			
			local reqprivs = {}
			local reqpriv = multiserver.get_conf_key("servers:" .. param .. ":priv")
			if reqpriv then
				reqprivs[reqpriv] = true
			end
			
			if not multiserver.check_player_privs(multiserver.get_player_name(id), reqprivs) then
				return "You do not have permission to join this server! Required privilege: " .. reqpriv
			end
			
			multiserver.redirect(id, param)
			return "Redirecting you to " .. param .. "."
		end
	end,
})

multiserver.register_chatcommand("find", {
	privs = {find = true},
	func = function(id, param)
		if not param or param == "" then
			return "Usage: /find <playername>"
		end
		
		local peerid = multiserver.get_peer_id(param)
		if peerid then
			return param .. " is connected to server " .. multiserver.get_current_server(peerid) .. "."
		else
			return param .. " is not online."
		end
	end,
})

multiserver.register_chatcommand("ip", {
	privs = {ip = true},
	func = function(id, param)
		if not param or param == "" then
			return "Usage: /ip <playername>"
		end
		
		local peerid = multiserver.get_peer_id(param)
		if peerid then
			local addr = multiserver.get_player_address(peerid)
			addr = multiserver.split(addr, ":")[1]
			return param .. "'s IP address is " .. addr
		else
			return param .. " is not online."
		end
	end,
})

multiserver.register_chatcommand("end", {
	privs = {end_proxy = true},
	func = function(id, param)
		multiserver.request_end(false)
	end,
})

multiserver.register_chatcommand("p_privs", {
	privs = {},
	func = function(id, param)
		local name = param
		if not name or name == "" then
			name = multiserver.get_player_name(id)
		end
		
		local privs = multiserver.get_player_privs(name)
		local privnames = {}
		for priv, v in pairs(privs) do
			table.insert(privnames, priv)
		end
		
		return name .. "'s privileges: " .. table.concat(privnames, " ")
	end,
})

multiserver.register_chatcommand("p_grant", {
	privs = {privs = true},
	func = function(id, param)
		local name = multiserver.split(param, " ")[1]
		local privnames = multiserver.split(param, " ")[2]
		
		if not privnames or privnames == "" then
			privnames = name
			name = multiserver.get_player_name(id)
		end
		
		if not multiserver.get_peer_id(name) then
			return name .. " is not online."
		end
		
		local privs = multiserver.get_player_privs(name)
		for _, newpriv in ipairs(multiserver.split(privnames:gsub(" ", ""), ",")) do
			privs[newpriv] = true
		end
		multiserver.set_player_privs(name, privs)
		
		return "Privileges updated."
	end,
})

multiserver.register_chatcommand("p_revoke", {
	privs = {privs = true},
	func = function(id, param)
		local name = multiserver.split(param, " ")[1]
		local privnames = multiserver.split(param, " ")[2]
		
		if not privnames or privnames == "" then
			privnames = name
			name = multiserver.get_player_name(id)
		end
		
		if not multiserver.get_peer_id(name) then
			return name .. " is not online."
		end
		
		local privs = multiserver.get_player_privs(name)
		for _, rmpriv in ipairs(multiserver.split(privnames:gsub(" ", ""), ",")) do
			privs[rmpriv] = nil
		end
		multiserver.set_player_privs(name, privs)
		
		return "Privileges updated."
	end,
})
