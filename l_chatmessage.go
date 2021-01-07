package multiserver

import (
	"strings"
	"encoding/binary"
	"time"
	"log"
	
	"github.com/yuin/gopher-lua"
)

type chatCommand struct {
	name     string
	function *lua.LFunction
}

var chatCommands []chatCommand
var chatMessageHandlers []*lua.LFunction

func registerChatCommand(L *lua.LState) int {
	name := L.ToString(1)
	cmddef := L.ToTable(2)
	f := cmddef.RawGet(lua.LString("func")).(*lua.LFunction)
	chatCommands = append(chatCommands, chatCommand{name: name, function: f})
	
	return 0
}

func registerOnChatMessage(L *lua.LState) int {
	f := L.ToFunction(1)
	chatMessageHandlers = append(chatMessageHandlers, f)
	
	return 0
}

func processChatMessage(peerid PeerID, msg []byte) bool {
	s := string(narrow(msg[4:]))
	if strings.HasPrefix(s, "/") {
		// Chat command
		s = strings.Replace(s, "/", "", 1)
		params := strings.Split(s, " ")
		for i := range chatCommands {
			if chatCommands[i].name == params[0] {
				if err := l.CallByParam(lua.P{Fn: chatCommands[i].function, NRet: 1, Protect: true}, lua.LNumber(peerid), lua.LString(strings.Join(params[1:], " "))); err != nil {
					log.Print(err)
					
					go func() {
						End(true, true)
					}()
				}
				if str, ok := l.Get(-1).(lua.LString); ok {
					wstr := wider([]byte(str.String()))
					
					data := make([]byte, 16 + len(wstr))
					data[0] = uint8(0x00)
					data[1] = uint8(0x2F)
					data[2] = uint8(0x01)
					data[3] = uint8(0x00)
					data[4] = uint8(0x00)
					data[5] = uint8(0x00)
					binary.BigEndian.PutUint16(data[6:8], uint16(len(str.String())))
					copy(data[8:8 + len(wstr)], wstr)
					data[8 + len(wstr)] = uint8(0x00)
					data[9 + len(wstr)] = uint8(0x00)
					data[10 + len(wstr)] = uint8(0x00)
					data[11 + len(wstr)] = uint8(0x00)
					binary.BigEndian.PutUint32(data[12 + len(wstr):16 + len(wstr)], uint32(time.Now().Unix()))
					
					ack, err := GetListener().GetPeerByID(peerid).Send(Pkt{Data: data, ChNo: 0, Unrel: false})
					if err != nil {
						log.Print(err)
					}
					<-ack
				}
				
				return true
			}
		}
	} else {
		// Regular message
		for i := range chatMessageHandlers {
			if err := l.CallByParam(lua.P{Fn: chatMessageHandlers[i], NRet: 1, Protect: true}, lua.LNumber(peerid), lua.LString(s)); err != nil {
				log.Print(err)
				
				End(true, true)
			}
			if b, ok := l.Get(-1).(lua.LBool); ok {
				if lua.LVAsBool(b) {
					return true
				}
			}
		}
	}
	
	return false
}

func chatSendPlayer(L *lua.LState) int {
	id := PeerID(L.ToInt(1))
	msg := L.ToString(2)
	l := GetListener()
	p := l.GetPeerByID(id)
	
	wstr := wider([]byte(msg))
	
	data := make([]byte, 16 + len(wstr))
	data[0] = uint8(0x00)
	data[1] = uint8(0x2F)
	data[2] = uint8(0x01)
	data[3] = uint8(0x00)
	data[4] = uint8(0x00)
	data[5] = uint8(0x00)
	binary.BigEndian.PutUint16(data[6:8], uint16(len(msg)))
	copy(data[8:8 + len(wstr)], wstr)
	data[8 + len(wstr)] = uint8(0x00)
	data[9 + len(wstr)] = uint8(0x00)
	data[10 + len(wstr)] = uint8(0x00)
	data[11 + len(wstr)] = uint8(0x00)
	binary.BigEndian.PutUint32(data[12 + len(wstr):16 + len(wstr)], uint32(time.Now().Unix()))
	
	ack, err := p.Send(Pkt{Data: data, ChNo: 0, Unrel: false})
	if err != nil {
		log.Print(err)
		return 0
	}
	<-ack
	
	return 0
}

func chatSendAll(L *lua.LState) int {
	msg := L.ToString(1)
	l := GetListener()
	
	wstr := wider([]byte(msg))
	
	data := make([]byte, 16 + len(wstr))
	data[0] = uint8(0x00)
	data[1] = uint8(0x2F)
	data[2] = uint8(0x01)
	data[3] = uint8(0x00)
	data[4] = uint8(0x00)
	data[5] = uint8(0x00)
	binary.BigEndian.PutUint16(data[6:8], uint16(len(msg)))
	copy(data[8:8 + len(wstr)], wstr)
	data[8 + len(wstr)] = uint8(0x00)
	data[9 + len(wstr)] = uint8(0x00)
	data[10 + len(wstr)] = uint8(0x00)
	data[11 + len(wstr)] = uint8(0x00)
	binary.BigEndian.PutUint32(data[12 + len(wstr):16 + len(wstr)], uint32(time.Now().Unix()))
	
	i := PeerIDCltMin
	for l.id2peer[i].Peer != nil {
		ack, err := l.id2peer[i].Send(Pkt{Data: data, ChNo: 0, Unrel: false})
		if err != nil {
			log.Print(err)
			return 0
		}
		<-ack
		
		i++
	}
	
	return 0
}

func narrow(b []byte) []byte {
	var r []byte
	for i := range b {
		if b[i] != uint8(0x00) {
			r = append(r, b[i])
		}
	}
	
	return r
}

func wider(b []byte) []byte {
	var r []byte
	for i := range b {
		r = append(r, uint8(0x00), b[i])
	}
	
	return r
}
