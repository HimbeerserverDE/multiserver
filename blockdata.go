package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"io"

	"github.com/anon55555/mt/rudp"
)

const NodeCount = 16 * 16 * 16

func processBlockdata(p *Peer, pkt *rudp.Pkt) bool {
	srv := p.ServerName()

	x := int16(binary.BigEndian.Uint16(pkt.Data[2:4]))
	y := int16(binary.BigEndian.Uint16(pkt.Data[4:6]))
	z := int16(binary.BigEndian.Uint16(pkt.Data[6:8]))

	p.blocks = append(p.blocks, [3]int16{x, y, z})

	r := bytes.NewReader(pkt.Data[13:])

	zr, err := zlib.NewReader(r)
	if err != nil {
		return true
	}

	buf := &bytes.Buffer{}
	_, err = io.Copy(buf, zr)
	if err != nil {
		return true
	}
	zr.Close()

	nodes := buf.Bytes()

	for i := uint32(0); i < NodeCount; i++ {
		contentID := binary.BigEndian.Uint16(nodes[2*i : 2+2*i])
		if contentID >= ContentUnknown && contentID <= ContentIgnore {
			continue
		}
		newID := nodeDefs[srv][contentID].ID()
		binary.BigEndian.PutUint16(nodes[2*i:2+2*i], newID)
	}

	var recompBuf bytes.Buffer
	zw := zlib.NewWriter(&recompBuf)
	zw.Write(nodes)
	zw.Close()

	recompNodes := recompBuf.Bytes()

	meta := make([]byte, 65536)
	n, err := r.Read(meta)
	if err != nil {
		return true
	}

	meta = meta[:n]

	data := make([]byte, 13+len(recompNodes)+len(meta))
	copy(data[:13], pkt.Data[:13])
	copy(data[13:13+len(recompNodes)], recompNodes)
	copy(data[13+len(recompNodes):], meta)

	pkt.Data = data

	return false
}

func processAddnode(p *Peer, pkt *rudp.Pkt) bool {
	srv := p.ServerName()

	contentID := binary.BigEndian.Uint16(pkt.Data[8:10])
	newID := nodeDefs[srv][contentID].ID()
	binary.BigEndian.PutUint16(pkt.Data[8:10], newID)

	return false
}
