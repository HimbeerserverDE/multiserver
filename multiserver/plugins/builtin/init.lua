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
