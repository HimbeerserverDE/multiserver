package multiserver

import (
	"github.com/yuin/gopher-lua"
	"log"
)

func getPlayerPrivs(L *lua.LState) int {
	name := L.ToString(1)

	r := L.NewTable()

	db, err := initAuthDB()
	if err != nil {
		log.Print(err)
		L.Push(lua.LNil)
		return 1
	}

	eprivs, err := readPrivItem(db, name)
	if err != nil {
		log.Print(err)
		L.Push(lua.LNil)
		return 1
	}

	db.Close()

	privs := decodePrivs(eprivs)

	for priv := range privs {
		if privs[priv] {
			r.RawSet(lua.LString(priv), lua.LBool(true))
		}
	}

	L.Push(r)

	return 1
}

func setPlayerPrivs(L *lua.LState) int {
	name := L.ToString(1)
	newprivs := L.ToTable(2)

	newpmap := make(map[string]bool)

	newprivs.ForEach(func(k, v lua.LValue) {
		if lua.LVAsBool(v) {
			newpmap[k.String()] = true
		}
	})

	ps := encodePrivs(newpmap)

	db, err := initAuthDB()
	if err != nil {
		log.Print(err)
		return 0
	}

	err = modPrivItem(db, name, ps)
	if err != nil {
		log.Print(err)
		return 0
	}

	return 0
}

func checkPlayerPrivs(L *lua.LState) int {
	name := L.ToString(1)
	reqprivs := L.ToTable(2)

	db, err := initAuthDB()
	if err != nil {
		log.Print(err)
		L.Push(lua.LBool(false))
		return 1
	}

	eprivs, err := readPrivItem(db, name)
	if err != nil {
		log.Print(err)
		L.Push(lua.LBool(false))
		return 1
	}

	db.Close()

	privs := decodePrivs(eprivs)

	hasPrivs := true
	reqprivs.ForEach(func(k, v lua.LValue) {
		if lua.LVAsBool(v) {
			if !privs[k.String()] {
				hasPrivs = false
			}
		}
	})

	if hasPrivs {
		L.Push(lua.LBool(true))
	} else {
		L.Push(lua.LBool(false))
	}

	return 1
}
