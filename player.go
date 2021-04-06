package main

import "sync"

const (
	MaxPlayerNameLength = 20
	PlayerNameChars     = "[a-zA-Z0-9-_]"
)

var onlinePlayers map[string]bool
var onlinePlayerMu sync.RWMutex

var onJoinPlayer []func(*Conn)
var onLeavePlayer []func(*Conn)

// RegisterOnJoinPlayer registers a callback function that is called
// when a TOSERVER_CLIENT_READY pkt is received from the Conn
func RegisterOnJoinPlayer(function func(*Conn)) {
	onJoinPlayer = append(onJoinPlayer, function)
}

// RegisterOnLeavePlayer registers a callback function that is called
// when a client Conn disconnects
func RegisterOnLeavePlayer(function func(*Conn)) {
	onLeavePlayer = append(onLeavePlayer, function)
}

func processJoin(c *Conn) {
	onlinePlayerMu.Lock()
	defer onlinePlayerMu.Unlock()

	cltSrv := c.ServerName()
	for ; cltSrv == ""; cltSrv = c.ServerName() {
	}

	rpcSrvMu.Lock()
	for srv := range rpcSrvs {
		srv.doRpc("->JOIN "+c.Username()+" "+cltSrv, "--")
	}
	rpcSrvMu.Unlock()

	onlinePlayers[c.Username()] = true
	for i := range onJoinPlayer {
		onJoinPlayer[i](c)
	}

	go OptimizeRPCConns()
}

func processLeave(c *Conn) {
	onlinePlayerMu.Lock()
	defer onlinePlayerMu.Unlock()

	rpcSrvMu.Lock()
	for srv := range rpcSrvs {
		srv.doRpc("->LEAVE "+c.Username(), "--")
	}
	rpcSrvMu.Unlock()

	onlinePlayers[c.Username()] = false
	for i := range onLeavePlayer {
		onLeavePlayer[i](c)
	}
}

// IsOnline reports if a player is connected
func IsOnline(name string) bool {
	onlinePlayerMu.RLock()
	defer onlinePlayerMu.RUnlock()

	return onlinePlayers[name]
}

func init() {
	onlinePlayers = make(map[string]bool)
}
