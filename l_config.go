package multiserver

import (
	"github.com/yuin/gopher-lua"
)

func luaGetConfKey(L *lua.LState) int {
	key := L.ToString(1)
	
	v := GetConfKey(key)
	
	L.Push(lua.LString(v.(string)))
	
	return 1
}
