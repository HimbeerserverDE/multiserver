package main

import (
	"encoding/binary"
	"log"

	"github.com/anon55555/mt/rudp"
)

func processAoRmAdd(p *Peer, data []byte) []byte {
	countRm := binary.BigEndian.Uint16(data[2:4])
	var aoRm []uint16
	for i := uint16(0); i < countRm; i += 2 {
		id := binary.BigEndian.Uint16(data[4+i:6+i])
		if id == p.localPlayerCao {
			id = p.currentPlayerCao
		}
		aoRm = append(aoRm, id)
	}

	countAdd := binary.BigEndian.Uint16(data[4+countRm*2 : 6+countRm*2])
	var aoAdd []uint16
	si := 6 + uint32(countRm)*2
	for i := uint32(0); i < uint32(countAdd); i++ {
		id := binary.BigEndian.Uint16(data[si:2+si])

		initDataLen := binary.BigEndian.Uint32(data[3+si : 7+si])

		namelen := binary.BigEndian.Uint16(data[8+si : 10+si])
		name := data[10+si : 10+si+uint32(namelen)]
		if string(name) == p.Username() {
			if p.initAoReceived {
				initData := data[7+si : 7+si+initDataLen]

				// Read the messages from the packet
				// They need to be forwarded
				msgcount := uint8(initData[32+namelen])
				var msgs [][]byte
				sj := uint16(33+namelen)
				for j := uint8(0); j < msgcount; j++ {
					msglen := binary.BigEndian.Uint16(initData[2+sj:4+sj])
					msg := initData[4+sj : 4+sj+msglen]
					msgs = append(msgs, msg)

					sj += 4 + msglen
				}

				// Generate message packet
				msgpkt := []byte{0x00, ToClientActiveObjectMessages}
				for _, msg := range msgs {
					msgdata := make([]byte, 4+len(msg))
					binary.BigEndian.PutUint16(msgdata[0:2], p.localPlayerCao)
					binary.BigEndian.PutUint16(msgdata[2:4], uint16(len(msg)))
					copy(msgdata[4:], msg)
					msgpkt = append(msgpkt, msgdata...)
				}

				ack, err := p.Send(rudp.Pkt{Data: msgpkt})
				if err != nil {
					log.Print(err)
				}
				<-ack

				binary.BigEndian.PutUint16(data[4+countRm*2:6+countRm*2], countAdd-1)
				data = append(data[:si], data[7+si+initDataLen:]...)
				p.currentPlayerCao = id
				si -= 7 + initDataLen
			} else {
				p.initAoReceived = true
				p.localPlayerCao = id
				p.currentPlayerCao = id
			}

			si += 7 + initDataLen
			continue
		} else if id == p.localPlayerCao {
			id = p.currentPlayerCao
			binary.BigEndian.PutUint16(data[si:2+si], id)
		}

		aoAdd = append(aoAdd, id)

		si += 7 + initDataLen
	}

	p.redirectMu.Lock()
	for i := range aoAdd {
		if aoAdd[i] != 0 {
			p.aoIDs[aoAdd[i]] = true
		}
	}

	for i := range aoRm {
		p.aoIDs[aoRm[i]] = false
	}
	p.redirectMu.Unlock()

	return data
}

func processAoMsgs(p *Peer, data []byte) []byte {
	si := uint32(2)
	for si < uint32(len(data)) {
		id := binary.BigEndian.Uint16(data[si:2+si])
		msglen := binary.BigEndian.Uint16(data[2+si : 4+si])

		if id == p.currentPlayerCao {
			id = p.localPlayerCao
			binary.BigEndian.PutUint16(data[si:2+si], id)
		} else if id == p.localPlayerCao {
			id = p.currentPlayerCao
			binary.BigEndian.PutUint16(data[si:2+si], id)
		}

		si += 2 + uint32(msglen)
	}
	return data
}
