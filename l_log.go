package multiserver

import (
	"github.com/yuin/gopher-lua"
	"log"
)

func luaLog(L *lua.LState) int {
	str := L.ToString(1)
	log.Print(str)
	return 0
}
