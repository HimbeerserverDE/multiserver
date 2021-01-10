package multiserver

import (
	"log"
	"time"
	"github.com/yuin/gopher-lua"
)

func luaAfter(L *lua.LState) int {
	t := L.ToInt(1)
	f := L.ToFunction(2)
	arg := L.CheckAny(3)

	go func() {
		time.Sleep(time.Duration(t) * time.Second)

		if err := L.CallByParam(lua.P{Fn: f, NRet: 0, Protect: true}, arg); err != nil {
			log.Print(err)

			End(true, true)
		}
	}()

	return 0
}
