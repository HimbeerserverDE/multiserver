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

multiserver.register_on_redirect_done(function(id, newsrv, success)
	if not success then
		multiserver.chat_send_player(id, "Could not connect you to " .. newsrv .. "!")
	else
		local name = multiserver.get_player_name(id)
		if name and name ~= "" then
			multiserver.set_storage_key("server:" .. name, newsrv)
		end
	end
end)

multiserver.register_on_joinplayer(function(id)
	if not multiserver.get_conf_key("force_default_server") then
		local name = multiserver.get_player_name(id)
		if not name or name == "" then return end
		local srv = multiserver.get_storage_key("server:" .. name)
		if not srv or srv == "" then
			srv = multiserver.get_conf_key("default_server")
		end
		multiserver.redirect(id, srv)
	end
end)
