package multiserver

import (
	"strings"
	
	"github.com/yuin/gopher-lua"
)

func luaStringSplit(L *lua.LState) {
	s := L.ToString(1)
	d := L.ToString(2)
	
	L.Push(lua.LString(strings.Split(s, d)))
	
	return 1
}
