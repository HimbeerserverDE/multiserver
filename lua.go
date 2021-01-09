package multiserver

import "github.com/yuin/gopher-lua"

var l *lua.LState
var api_funcs *lua.LTable

func InitLua() {
	l = lua.NewState()
	
	api_funcs = l.NewTable()
	l.SetGlobal("multiserver", api_funcs)
	
	// redirect
	addLuaFunc(redirect, "redirect")
	addLuaFunc(getServers, "get_servers")
	addLuaFunc(registerOnRedirectDone, "register_on_redirect_done")
	// chatmessage
	addLuaFunc(registerChatCommand, "register_chatcommand")
	addLuaFunc(registerOnChatMessage, "register_on_chatmessage")
	addLuaFunc(chatSendPlayer, "chat_send_player")
	addLuaFunc(chatSendAll, "chat_send_all")
	// player
	addLuaFunc(getPlayerName, "get_player_name")
	addLuaFunc(luaGetPeerID, "get_peer_id")
	addLuaFunc(kickPlayer, "kick_player")
	addLuaFunc(getCurrentServer, "get_current_server")
	addLuaFunc(getPlayerAddress, "get_player_address")
	addLuaFunc(getConnectedPlayers, "get_connected_players")
	addLuaFunc(registerOnJoinPlayer, "register_on_joinplayer")
	addLuaFunc(registerOnLeavePlayer, "register_on_leaveplayer")
	// log
	addLuaFunc(luaLog, "log")
	// string
	addLuaFunc(luaStringSplit, "split")
	// end
	addLuaFunc(luaEnd, "request_end")
	// privileges
	addLuaFunc(getPlayerPrivs, "get_player_privs")
	addLuaFunc(setPlayerPrivs, "set_player_privs")
	addLuaFunc(checkPlayerPrivs, "check_player_privs")
	// config
	addLuaFunc(luaGetConfKey, "get_conf_key")
}

func CloseLua() {
	l.Close()
}

func addLuaFunc(f func(*lua.LState) int, name string) {
	api_funcs.RawSet(lua.LString(name), l.NewFunction(f))
}
