package main

import "sync"

var onlinePlayers map[string]bool
var onlinePlayerMu sync.RWMutex

var onJoinPlayer []func(*Peer)
var onLeavePlayer []func(*Peer)

// RegisterOnJoinPlayer registers a callback function that is called
// when a TOSERVER_CLIENT_READY pkt is received from the Peer
func RegisterOnJoinPlayer(function func(*Peer)) {
	onJoinPlayer = append(onJoinPlayer, function)
}

// RegisterOnLeavePlayer registers a callback function that is called
// when a client Peer disconnects
func RegisterOnLeavePlayer(function func(*Peer)) {
	onLeavePlayer = append(onLeavePlayer, function)
}

func processJoin(p *Peer) {
	onlinePlayerMu.Lock()
	defer onlinePlayerMu.Unlock()

	rpcSrvMu.Lock()
	for srv := range rpcSrvs {
		srv.doRpc("->JOIN "+string(p.username), "--")
	}
	rpcSrvMu.Unlock()

	onlinePlayers[string(p.username)] = true
	for i := range onJoinPlayer {
		onJoinPlayer[i](p)
	}
}

func processLeave(p *Peer) {
	onlinePlayerMu.Lock()
	defer onlinePlayerMu.Unlock()

	rpcSrvMu.Lock()
	for srv := range rpcSrvs {
		srv.doRpc("->LEAVE "+string(p.username), "--")
	}
	rpcSrvMu.Unlock()

	onlinePlayers[string(p.username)] = false
	for i := range onLeavePlayer {
		onLeavePlayer[i](p)
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
