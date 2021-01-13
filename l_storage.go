package multiserver

import (
	"github.com/yuin/gopher-lua"
	"log"
)

func setStorageKey(L *lua.LState) int {
	k := L.ToString(1)
	v := L.ToString(2)

	db, err := initPluginStorageDB()
	if err != nil {
		log.Print(err)
		return 0
	}

	if k != "" {
		err = modOrAddPluginStorageItem(db, k, v)
		if err != nil {
			log.Print(err)
			return 0
		}
	} else {
		err = deletePluginStorageItem(db, k)
		if err != nil {
			log.Print(err)
			return 0
		}
	}

	return 0
}

func getStorageKey(L *lua.LState) int {
	k := L.ToString(1)

	db, err := initPluginStorageDB()
	if err != nil {
		log.Print(err)
		L.Push(lua.LString(""))
		return 1
	}

	v, err := readPluginStorageItem(db, k)
	if err != nil {
		log.Print(err)
		L.Push(lua.LString(""))
		return 1
	}

	L.Push(lua.LString(v))

	return 1
}
