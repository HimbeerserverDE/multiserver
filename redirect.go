package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"log"
	"math"
	"net"

	"github.com/anon55555/mt/rudp"
)

var onRedirectDone []func(*Conn, string, bool)

// RegisterOnRedirectDone registers a callback function that is called
// when the Conn.Redirect method exits
func RegisterOnRedirectDone(function func(*Conn, string, bool)) {
	onRedirectDone = append(onRedirectDone, function)
}

func processRedirectDone(c *Conn, newsrv *string) {
	success := c.ServerName() == *newsrv

	successstr := "false"
	if success {
		successstr = "true"
	}

	rpcSrvMu.Lock()
	for srv := range rpcSrvs {
		srv.doRPC("->REDIRECTED "+c.Username()+" "+*newsrv+" "+successstr, "--")
	}
	rpcSrvMu.Unlock()

	for i := range onRedirectDone {
		onRedirectDone[i](c, *newsrv, success)
	}
}

// Redirect sends the Conn to the minetest server named newsrv
func (c *Conn) Redirect(newsrv string) error {
	c.redirectMu.Lock()
	defer c.redirectMu.Unlock()

	defer processRedirectDone(c, &newsrv)

	straddr, ok := ConfKey("servers:" + newsrv + ":address").(string)
	if !ok {
		grp, ok := ConfKey("groups:" + newsrv).([]interface{})
		if !ok {
			return fmt.Errorf("server or group %s does not exist", newsrv)
		}

		smallestCnt := int(^uint(0) >> 1)
		for _, srv := range grp {
			cnt := len(ConnsServer(srv.(string)))
			if cnt < smallestCnt {
				if c.ServerName() == srv.(string) {
					return fmt.Errorf("already connected to server %s", srv.(string))
				}

				smallestCnt = cnt
				newsrv = srv.(string)
			}
		}

		straddr, ok = ConfKey("servers:" + newsrv + ":address").(string)
		if !ok {
			return fmt.Errorf("server %s does not exist", newsrv)
		}
	}

	if c.ServerName() == newsrv {
		return fmt.Errorf("already connected to server %s", newsrv)
	}

	srvaddr, err := net.ResolveUDPAddr("udp", straddr)
	if err != nil {
		return err
	}

	conn, err := net.DialUDP("udp", nil, srvaddr)
	if err != nil {
		return err
	}

	srv, err := Connect(conn)
	if err != nil {
		return err
	}

	fin := make(chan *Conn)
	go Init(c, srv, true, false, fin)
	initOk := <-fin

	if initOk == nil {
		srv.Close()
		return fmt.Errorf("initialization with server %s failed", newsrv)
	}

	// Reset formspec style
	data := []byte{
		0x00, ToClientFormspecPrepend,
		0x00, 0x00,
	}

	_, err = c.Send(rudp.Pkt{Reader: bytes.NewReader(data)})

	// Remove active objects
	data = make([]byte, 6+len(c.aoIDs)*2)
	data[0] = uint8(0x00)
	data[1] = uint8(ToClientActiveObjectRemoveAdd)
	binary.BigEndian.PutUint16(data[2:4], uint16(len(c.aoIDs)))

	si := 4
	for ao := range c.aoIDs {
		binary.BigEndian.PutUint16(data[si:2+si], ao)
		si += 2
	}

	binary.BigEndian.PutUint16(data[si:2+si], uint16(0))

	_, err = c.Send(rudp.Pkt{Reader: bytes.NewReader(data)})
	if err != nil {
		return err
	}

	c.aoIDs = make(map[uint16]bool)

	// Remove MapBlocks
	for _, block := range c.blocks {
		x := block[0]
		y := block[1]
		z := block[2]

		blockdata := make([]byte, 5)
		blockdata[0] = uint8(0)
		binary.BigEndian.PutUint16(blockdata[1:3], uint16(0xFFFF))
		blockdata[3] = uint8(2)
		blockdata[4] = uint8(2)

		nodes := make([]byte, 16384)
		for i := uint32(0); i < NodeCount; i++ {
			binary.BigEndian.PutUint16(nodes[2*i:2+2*i], uint16(ContentIgnore))
			nodes[2*NodeCount+i] = uint8(0)
			nodes[3*NodeCount+i] = uint8(0)
		}

		var compBuf bytes.Buffer
		zw := zlib.NewWriter(&compBuf)
		zw.Write(nodes)
		zw.Close()

		compNodes := compBuf.Bytes()

		w := bytes.NewBuffer([]byte{0x00, ToClientBlockdata})
		WriteUint16(w, uint16(x))
		WriteUint16(w, uint16(y))
		WriteUint16(w, uint16(z))
		w.Write(blockdata)
		w.Write(compNodes)

		_, err = c.Send(rudp.Pkt{Reader: w})
		if err != nil {
			return err
		}
	}

	c.blocks = [][3]int16{}

	// Remove HUDs
	data = []byte{0, ToClientHudSetParam, 0, 1, 0, 4, 0, 0, 0, 8}

	_, err = c.Send(rudp.Pkt{
		Reader: bytes.NewReader(data),
		PktInfo: rudp.PktInfo{
			Channel: 1,
		},
	})

	if err != nil {
		return err
	}

	data = []byte{0, ToClientHudSetParam, 0, 2, 0, 0}

	_, err = c.Send(rudp.Pkt{
		Reader: bytes.NewReader(data),
		PktInfo: rudp.PktInfo{
			Channel: 1,
		},
	})

	if err != nil {
		return err
	}

	data = []byte{0, ToClientHudSetParam, 0, 3, 0, 0}

	_, err = c.Send(rudp.Pkt{
		Reader: bytes.NewReader(data),
		PktInfo: rudp.PktInfo{
			Channel: 1,
		},
	})

	if err != nil {
		return err
	}

	for hud := range c.huds {
		data = make([]byte, 6)
		data[0] = uint8(0x00)
		data[1] = uint8(ToClientHudRM)
		binary.BigEndian.PutUint32(data[2:6], hud)

		_, err = c.Send(rudp.Pkt{
			Reader: bytes.NewReader(data),
			PktInfo: rudp.PktInfo{
				Channel: 1,
			},
		})

		if err != nil {
			return err
		}
	}

	c.huds = make(map[uint32]bool)

	// Stop looped sounds
	for sound := range c.sounds {
		data = make([]byte, 6)
		data[0] = uint8(0x00)
		data[1] = uint8(ToClientStopSound)
		binary.BigEndian.PutUint32(data[2:6], uint32(sound))

		_, err = c.Send(rudp.Pkt{Reader: bytes.NewReader(data)})
		if err != nil {
			return err
		}
	}

	c.sounds = make(map[int32]bool)

	// Stop day/night ratio override
	data = []byte{0, 0, 0}

	_, err = c.Send(rudp.Pkt{Reader: bytes.NewReader(data)})
	if err != nil {
		return err
	}

	// Reset eye offset
	data = []byte{}
	for i := 0; i < 24; i++ {
		data = append(data, uint8(0))
	}

	_, err = c.Send(rudp.Pkt{Reader: bytes.NewReader(data)})
	if err != nil {
		return err
	}

	// Reset sky
	switch c.ProtoVer() {
	case 39:
		data = []byte{
			0, ToClientSetSky,
			0, 0, 0, 0,
			0, 7, 114, 101, 103, 117, 108, 97, 114,
			1,
			255, 255, 255, 255,
			255, 255, 255, 255,
			0, 7, 100, 101, 102, 97, 117, 108, 116,
			255, 97, 181, 245,
			255, 144, 211, 245,
			255, 180, 186, 250,
			255, 186, 193, 240,
			255, 0, 107, 255,
			255, 64, 144, 255,
			255, 100, 100, 100,
		}
	default:
		data = []byte{
			0, ToClientSetSky,
			0, 0, 0, 0,
			0, 7, 114, 101, 103, 117, 108, 97, 114,
			0, 0,
		}
	}

	_, err = c.Send(rudp.Pkt{Reader: bytes.NewReader(data)})
	if err != nil {
		return err
	}

	// Reset sun
	data = []byte{
		1,
		0, 7, 115, 117, 110, 46, 112, 110, 103,
		0, 15, 115, 117, 110, 95, 116, 111, 110, 101, 109, 97, 112, 46, 112, 110, 103,
		0, 13, 115, 117, 110, 114, 105, 115, 101, 98, 103, 46, 112, 110, 103,
	}
	sunscale := make([]byte, 4)
	binary.BigEndian.PutUint32(sunscale[0:4], math.Float32bits(1))
	data = append(data, sunscale...)

	_, err = c.Send(rudp.Pkt{Reader: bytes.NewReader(data)})
	if err != nil {
		return err
	}

	// Reset moon
	data = []byte{
		1,
		0, 8, 109, 111, 111, 110, 46, 112, 110, 103,
		0, 16, 109, 111, 111, 110, 95, 116, 111, 110, 101, 109, 97, 112, 46, 112, 110, 103,
	}
	moonscale := make([]byte, 4)
	binary.BigEndian.PutUint32(moonscale, math.Float32bits(1))
	data = append(data, moonscale...)

	_, err = c.Send(rudp.Pkt{Reader: bytes.NewReader(data)})
	if err != nil {
		return err
	}

	// Reset stars
	data = []byte{
		1,
		0, 0, 3, 232,
		105, 235, 235, 255,
	}
	starscale := make([]byte, 4)
	binary.BigEndian.PutUint32(starscale, math.Float32bits(1))
	data = append(data, starscale...)

	_, err = c.Send(rudp.Pkt{Reader: bytes.NewReader(data)})
	if err != nil {
		return err
	}

	// Reset cloud params
	w := bytes.NewBuffer([]byte{0x00, ToClientCloudParams})
	WriteUint32(w, math.Float32bits(0))
	w.Write([]byte{0, 0, 0, 0, 0, 0, 0, 0})
	WriteUint32(w, math.Float32bits(0))
	WriteUint32(w, math.Float32bits(0))
	WriteUint32(w, math.Float32bits(0))
	WriteUint32(w, math.Float32bits(0))

	_, err = c.Send(rudp.Pkt{Reader: bytes.NewReader(data)})
	if err != nil {
		return err
	}

	// Update detached inventories
	if len(detachedinvs[newsrv]) > 0 {
		for i := range detachedinvs[newsrv] {
			data = make([]byte, 2+len(detachedinvs[newsrv][i]))
			data[0] = uint8(0x00)
			data[1] = uint8(ToClientDetachedInventory)
			copy(data[2:], detachedinvs[newsrv][i])

			_, err = c.Send(rudp.Pkt{Reader: bytes.NewReader(data)})
			if err != nil {
				return err
			}
		}
	}

	c.Server().stopForwarding()

	c.SetServer(srv)

	go Proxy(c, srv)
	go Proxy(srv, c)

	// Rejoin mod channels
	for ch := range c.modChs {
		data := make([]byte, 4+len(ch))
		data[0] = uint8(0x00)
		data[1] = uint8(ToServerModChannelJoin)
		binary.BigEndian.PutUint16(data[2:4], uint16(len(ch)))
		copy(data[4:], []byte(ch))

		_, err = srv.Send(rudp.Pkt{Reader: bytes.NewReader(data)})
		if err != nil {
			log.Print(err)
		}
	}

	log.Print(c.Addr().String() + " redirected to " + newsrv)

	return nil
}
