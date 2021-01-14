package multiserver

import "sync"

var onlinePlayers map[string]bool
var onlinePlayerMu sync.RWMutex

var onJoinPlayer []func(*Peer)
var onLeavePlayer []func(*Peer)

func RegisterOnJoinPlayer(function func(*Peer)) {
	onJoinPlayer = append(onJoinPlayer, function)
}

func RegisterOnLeavePlayer(function func(*Peer)) {
	onLeavePlayer = append(onLeavePlayer, function)
}

func processJoin(p *Peer) {
	onlinePlayerMu.Lock()
	defer onlinePlayerMu.Unlock()

	onlinePlayers[string(p.username)] = true
	for i := range onJoinPlayer {
		onJoinPlayer[i](p)
	}
}

func processLeave(p *Peer) {
	onlinePlayerMu.Lock()
	defer onlinePlayerMu.Unlock()

	onlinePlayers[string(p.username)] = false
	for i := range onLeavePlayer {
		onLeavePlayer[i](p)
	}
}

func IsOnline(name string) bool {
	onlinePlayerMu.RLock()
	defer onlinePlayerMu.RUnlock()

	return onlinePlayers[name]
}

func init() {
	onlinePlayers = make(map[string]bool)
}
