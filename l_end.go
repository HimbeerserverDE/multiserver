package multiserver

import "github.com/yuin/gopher-lua"

func luaEnd(L *lua.LState) int {
	reconnect := L.ToBool(1)

	go func() {
		End(false, reconnect)
	}()

	return 0
}
