package multiserver

import (
	"github.com/yuin/gopher-lua"
)

func luaGetConfKey(L *lua.LState) int {
	key := L.ToString(1)

	v := GetConfKey(key)

	if v == nil {
		L.Push(lua.LNil)
	} else {
		L.Push(lua.LString(v.(string)))
	}

	return 1
}
