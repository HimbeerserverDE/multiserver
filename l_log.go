package multiserver

import (
	"log"
	"github.com/yuin/gopher-lua"
)

func luaLog(L *lua.LState) int {
	str := L.ToString(1)
	log.Print(str)

	return 0
}
