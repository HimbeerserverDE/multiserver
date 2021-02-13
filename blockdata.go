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

	si := 14
	// Check for zlib header
	for ; !(pkt.Data[si] == 120 && (pkt.Data[1+si] == 0x01 || pkt.Data[1+si] == 0x9C || pkt.Data[1+si] == 0xDA)); si++ {
	}

	compressedNodes := pkt.Data[13:si]

	zr, err := zlib.NewReader(bytes.NewReader(compressedNodes))
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

	data := make([]byte, 13+len(recompNodes)+len(pkt.Data[si:]))
	copy(data[:13], pkt.Data[:13])
	copy(data[13:13+len(recompNodes)], recompNodes)
	copy(data[13+len(recompNodes):], pkt.Data[si:])

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
