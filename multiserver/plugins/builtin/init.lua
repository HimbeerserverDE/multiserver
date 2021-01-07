multiserver.register_chatcommand("send", {
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
	func = function(id, param)
		multiserver.chat_send_all("[ALERT] " .. param)
	end,
})

multiserver.register_chatcommand("server", {
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
			
			multiserver.redirect(id, param)
			return "Redirecting you to " .. param .. "."
		end
	end,
})

multiserver.register_chatcommand("find", {
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
	func = function(id, param)
		multiserver.request_end(false)
	end,
})
