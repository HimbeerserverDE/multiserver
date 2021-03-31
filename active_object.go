package main

import (
	"bytes"
	"encoding/binary"
	"io"
	"log"

	"github.com/anon55555/mt/rudp"
)

const (
	AoCmdSetProps = iota
	AoCmdUpdatePos
	AoCmdSetTextureMod
	AoCmdSetSprite
	AoCmdPunched
	AoCmdUpdateArmorGroups
	AoCmdSetAnimation
	AoCmdSetBonePos
	AoCmdAttachTo
	AoCmdSetPhysicsOverride
	AoCmdObsolete1
	AoCmdSpawnInfant
	AoCmdSetAnimSpeed
)

func processAoRmAdd(c *Conn, r *bytes.Reader) []byte {
	w := &bytes.Buffer{}

	countRmBytes := make([]byte, 2)
	r.Read(countRmBytes)
	w.Write(countRmBytes)
	countRm := binary.BigEndian.Uint16(countRmBytes)

	var aoRm []uint16
	for i := uint16(0); i < countRm; i++ {
		idBytes := make([]byte, 2)
		r.Read(idBytes)
		w.Write(idBytes)
		id := binary.BigEndian.Uint16(idBytes)

		if id == c.localPlayerCao {
			id = c.currentPlayerCao
		}
		aoRm = append(aoRm, id)
	}

	countAddBytes := make([]byte, 2)
	r.Read(countAddBytes)
	w.Write(countAddBytes)
	countAdd := binary.BigEndian.Uint16(countAddBytes)

	var aoAdd []uint16
	for i := uint16(0); i < countAdd; i++ {
		idBytes := make([]byte, 2)
		r.Read(idBytes)
		id := binary.BigEndian.Uint16(idBytes)

		typeByte, _ := r.ReadByte()

		initDataLenBytes := make([]byte, 4)
		r.Read(initDataLenBytes)
		initDataLen := binary.BigEndian.Uint32(initDataLenBytes)

		initData := make([]byte, initDataLen)
		r.Read(initData)

		dr := bytes.NewReader(initData)

		dr.Seek(1, io.SeekStart)

		namelenBytes := make([]byte, 2)
		dr.Read(namelenBytes)
		namelen := binary.BigEndian.Uint16(namelenBytes)

		name := make([]byte, namelen)
		dr.Read(name)

		if string(name) == c.Username() {
			if c.initAoReceived {
				// Read the messages from the packet
				// They need to be forwarded
				dr.Seek(30, io.SeekCurrent)

				msgcountByte, _ := dr.ReadByte()
				msgcount := uint8(msgcountByte)

				var msgs [][]byte
				for j := uint8(0); j < msgcount; j++ {
					dr.Seek(2, io.SeekCurrent)

					msglenBytes := make([]byte, 2)
					dr.Read(msglenBytes)
					msglen := binary.BigEndian.Uint16(msglenBytes)

					msg := make([]byte, msglen)
					dr.Read(msg)

					msgs = append(msgs, msg)
				}

				// Generate message packet
				msgpkt := []byte{0x00, ToClientActiveObjectMessages}
				for _, msg := range msgs {
					msgdata := make([]byte, 4+len(msg))
					binary.BigEndian.PutUint16(msgdata[0:2], c.localPlayerCao)
					binary.BigEndian.PutUint16(msgdata[2:4], uint16(len(msg)))
					copy(msgdata[4:], aoMsgReplaceIDs(c, msg))
					msgpkt = append(msgpkt, msgdata...)
				}

				ack, err := c.Send(rudp.Pkt{Reader: bytes.NewReader(msgpkt)})
				if err != nil {
					log.Print(err)
				}
				<-ack

				data := w.Bytes()
				binary.BigEndian.PutUint16(data[4+countRm*2:6+countRm*2], countAdd-1)
				w = bytes.NewBuffer(data)

				c.currentPlayerCao = id
				continue
			} else {
				c.initAoReceived = true
				c.localPlayerCao = id
				c.currentPlayerCao = id
			}
		} else if id == c.localPlayerCao {
			id = c.currentPlayerCao
		}

		if string(name) != c.Username() {
			aoAdd = append(aoAdd, id)
		}

		binary.BigEndian.PutUint16(idBytes, id)
		w.Write(idBytes)
		w.WriteByte(typeByte)
		w.Write(initDataLenBytes)
		w.Write(initData)
	}

	c.redirectMu.Lock()
	for i := range aoAdd {
		if aoAdd[i] != 0 {
			c.aoIDs[aoAdd[i]] = true
		}
	}

	for i := range aoRm {
		c.aoIDs[aoRm[i]] = false
	}
	c.redirectMu.Unlock()

	return w.Bytes()
}

func processAoMsgs(c *Conn, r *bytes.Reader) []byte {
	w := &bytes.Buffer{}

	for r.Len() >= 4 {
		idBytes := make([]byte, 2)
		r.Read(idBytes)
		id := binary.BigEndian.Uint16(idBytes)

		msglenBytes := make([]byte, 2)
		r.Read(msglenBytes)
		msglen := binary.BigEndian.Uint16(msglenBytes)

		msg := make([]byte, msglen)
		r.Read(msg)

		msg = aoMsgReplaceIDs(c, msg)

		if id == c.currentPlayerCao {
			id = c.localPlayerCao
		} else if id == c.localPlayerCao {
			id = c.currentPlayerCao
		}

		binary.BigEndian.PutUint16(idBytes, id)
		w.Write(idBytes)
		w.Write(msglenBytes)
		w.Write(msg)
	}

	return w.Bytes()
}

func aoMsgReplaceIDs(c *Conn, data []byte) []byte {
	switch cmd := data[0]; cmd {
	case AoCmdAttachTo:
		id := binary.BigEndian.Uint16(data[1:3])
		if id == c.currentPlayerCao {
			id = c.localPlayerCao
			binary.BigEndian.PutUint16(data[1:3], id)
		} else if id == c.localPlayerCao {
			id = c.currentPlayerCao
			binary.BigEndian.PutUint16(data[1:3], id)
		}
	case AoCmdSpawnInfant:
		id := binary.BigEndian.Uint16(data[1:3])
		if id == c.currentPlayerCao {
			id = c.localPlayerCao
			binary.BigEndian.PutUint16(data[1:3], id)
		} else if id == c.localPlayerCao {
			id = c.currentPlayerCao
			binary.BigEndian.PutUint16(data[1:3], id)
		}
	}

	return data
}
