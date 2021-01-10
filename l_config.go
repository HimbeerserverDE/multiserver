package multiserver

import "github.com/yuin/gopher-lua"

func luaGetConfKey(L *lua.LState) int {
	key := L.ToString(1)

	v := GetConfKey(key)

	if v == nil {
		L.Push(lua.LNil)
	} else {
		switch v.(type) {
		case bool:
			L.Push(lua.LBool(v.(bool)))
		case int:
			L.Push(lua.LNumber(v.(int)))
		case string:
			L.Push(lua.LString(v.(string)))
		}
	}

	return 1
}
