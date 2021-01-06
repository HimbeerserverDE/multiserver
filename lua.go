package multiserver

import (
	"github.com/yuin/gopher-lua"
)

var l *lua.LState
var api_funcs *lua.LTable

func InitLua() {
	l = lua.NewState()
	
	api_funcs = l.NewTable()
	l.SetGlobal("multiserver", api_funcs)
	
	addLuaFunc(redirect, "redirect")
	addLuaFunc(registerChatCommand, "register_chatcommand")
	addLuaFunc(chatSendPlayer, "chat_send_player")
	addLuaFunc(chatSendAll, "chat_send_all")
	addLuaFunc(getPlayerName, "get_player_name")
}

func CloseLua() {
	l.Close()
}

func addLuaFunc(f func(*lua.LState) int, name string) {
	api_funcs.RawSet(lua.LString(name), l.NewFunction(f))
}
