package multiserver

import (
	"encoding/binary"
	"log"
	"strings"
	"time"
	"unicode/utf16"
)

type chatCommand struct {
	privs    map[string]bool
	function func(*Peer, string)
}

var chatCommands map[string]chatCommand
var onChatMsg    []func(*Peer, string) bool

func RegisterChatCommand(name string, privs map[string]bool, function func(*Peer, string)) {
	chatCommands[name] = chatCommand{privs: privs, function: function}
}

func registerOnChatMessage(function func(*Peer, string) bool) {
	onChatMsg = append(onChatMsg, function)
}

func processChatMessage(p *Peer, pkt Pkt) bool {
	s := string(narrow(pkt.Data[4:]))
	if strings.HasPrefix(s, "#") {
		// Chat command
		s = strings.Replace(s, "#", "", 1)
		params := strings.Split(s, " ")

		// Priv check
		allow, err := p.checkPrivs(chatCommands[params[0]].privs)
		if err != nil {
			log.Print(err)
			return true
		}

		if !allow {
			str := "You do not have permission to run this command! Required privileges: " + strings.Replace(encodePrivs(chatCommands[params[0]].privs), "|", " ", -1)
			wstr := wider([]byte(str))

			data := make([]byte, 16+len(wstr))
			data[0] = uint8(0x00)
			data[1] = uint8(ToClientChatMessage)
			data[2] = uint8(0x01)
			data[3] = uint8(0x00)
			data[4] = uint8(0x00)
			data[5] = uint8(0x00)
			binary.BigEndian.PutUint16(data[6:8], uint16(len(str)))
			copy(data[8:8+len(wstr)], wstr)
			data[8+len(wstr)] = uint8(0x00)
			data[9+len(wstr)] = uint8(0x00)
			data[10+len(wstr)] = uint8(0x00)
			data[11+len(wstr)] = uint8(0x00)
			binary.BigEndian.PutUint32(data[12+len(wstr):16+len(wstr)], uint32(time.Now().Unix()))

			ack, err := p.Send(Pkt{Data: data})
			if err != nil {
				log.Print(err)
			}
			<-ack

			return true
		}

		// Callback
		chatCommands[params[0]].function(p, strings.Join(params[1:], " "))
		return true
	} else {
		// Regular message
		forward := true
		for i := range onChatMsg {
			if onChatMsg[i](p, s) {
				forward = false
			}
		}
		return forward
	}
}

func (p *Peer) SendChatMsg(msg string) {
	wstr := wider([]byte(msg))

	data := make([]byte, 16+len(wstr))
	data[0] = uint8(0x00)
	data[1] = uint8(ToClientChatMessage)
	data[2] = uint8(0x01)
	data[3] = uint8(0x00)
	data[4] = uint8(0x00)
	data[5] = uint8(0x00)
	binary.BigEndian.PutUint16(data[6:8], uint16(len(msg)))
	copy(data[8:8+len(wstr)], wstr)
	data[8+len(wstr)] = uint8(0x00)
	data[9+len(wstr)] = uint8(0x00)
	data[10+len(wstr)] = uint8(0x00)
	data[11+len(wstr)] = uint8(0x00)
	binary.BigEndian.PutUint32(data[12+len(wstr):16+len(wstr)], uint32(time.Now().Unix()))

	ack, err := p.Send(Pkt{Data: data})
	if err != nil {
		log.Print(err)
	}
	<-ack
}

func ChatSendAll(msg string) {
	l := GetListener()
	l.mu.Lock()
	defer l.mu.Unlock()

	for i := range l.addr2peer {
		l.addr2peer[i].sendChatMsg(msg)
	}
}

func narrow(b []byte) []byte {
	if len(b)%2 != 0 {
		return nil
	}

	e := make([]uint16, len(b)/2)

	for i := 0; i < len(b); i += 2 {
		e[i/2] = binary.BigEndian.Uint16(b[i : 2+i])
	}

	return []byte(string(utf16.Decode(e)))
}

func wider(b []byte) []byte {
	r := make([]byte, len(b)*2)

	e := utf16.Encode([]rune(string(b)))

	for i := range e {
		binary.BigEndian.PutUint16(r[i*2:2+i*2], e[i])
	}

	return r
}
