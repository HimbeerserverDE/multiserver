package multiserver

import (
	"strings"
	
	"github.com/yuin/gopher-lua"
)

func luaStringSplit(L *lua.LState) int {
	s := L.ToString(1)
	d := L.ToString(2)
	
	L.Push(lua.LString(strings.Join(strings.Split(s, d), "")))
	
	return 1
}
