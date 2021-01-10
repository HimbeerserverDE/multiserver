package multiserver

import (
	"strings"
	"github.com/yuin/gopher-lua"
)

func luaStringSplit(L *lua.LState) int {
	s := L.ToString(1)
	d := L.ToString(2)

	split := strings.Split(s, d)
	r := L.NewTable()
	for i := range split {
		r.Append(lua.LString(split[i]))
	}

	L.Push(r)

	return 1
}
