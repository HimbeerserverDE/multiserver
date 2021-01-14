package multiserver

var onJoinPlayer []func(*Peer)
var onLeavePlayer []func(*Peer)

func RegisterOnJoinPlayer(function func(*Peer)) {
	onJoinPlayer = append(onJoinPlayer, function)
}

func RegisterOnLeavePlayer(function func(*Peer)) {
	onLeavePlayer = append(onLeavePlayer, function)
}

func processJoin(p *Peer) {
	for i := range onJoinPlayer {
		onJoinPlayer[i](p)
	}
}

func processLeave(p *Peer) {
	for i := range onLeavePlayer {
		onLeavePlayer[i](p)
	}
}
