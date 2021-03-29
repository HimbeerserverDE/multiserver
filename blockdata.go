package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"io"
)

const NodeCount = 16 * 16 * 16

func processBlockdata(c *Conn, r *bytes.Reader) ([]byte, bool) {
	srv := c.ServerName()

	posData := make([]byte, 6)
	r.Read(posData)

	x := int16(binary.BigEndian.Uint16(posData[0:2]))
	y := int16(binary.BigEndian.Uint16(posData[2:4]))
	z := int16(binary.BigEndian.Uint16(posData[4:6]))

	c.blocks = append(c.blocks, [3]int16{x, y, z})

	r.Seek(13, io.SeekStart)

	zr, err := zlib.NewReader(r)
	if err != nil {
		return nil, true
	}

	buf := &bytes.Buffer{}
	_, err = io.Copy(buf, zr)
	if err != nil {
		return nil, true
	}
	zr.Close()

	nodes := buf.Bytes()

	for i := uint32(0); i < NodeCount; i++ {
		contentID := binary.BigEndian.Uint16(nodes[2*i : 2+2*i])
		if contentID >= ContentUnknown && contentID <= ContentIgnore {
			continue
		}
		newID := NodeDefs()[srv][contentID].ID()
		binary.BigEndian.PutUint16(nodes[2*i:2+2*i], newID)
	}

	var recompBuf bytes.Buffer
	zw := zlib.NewWriter(&recompBuf)
	zw.Write(nodes)
	zw.Close()

	recompNodes := recompBuf.Bytes()

	meta := make([]byte, r.Len())
	r.Read(meta)

	r.Seek(2, io.SeekStart)

	blockMeta := make([]byte, 11)
	r.Read(blockMeta)

	data := make([]byte, 11+len(recompNodes)+len(meta))
	copy(data[:11], blockMeta)
	copy(data[11:11+len(recompNodes)], recompNodes)
	copy(data[11+len(recompNodes):], meta)

	return data, false
}

func processAddnode(c *Conn, r *bytes.Reader) []byte {
	srv := c.ServerName()

	r.Seek(8, io.SeekStart)

	idBytes := make([]byte, 2)
	r.Read(idBytes)

	contentID := binary.BigEndian.Uint16(idBytes)
	newID := NodeDefs()[srv][contentID].ID()

	r.Seek(2, io.SeekStart)

	data := make([]byte, r.Len())
	r.Read(data)

	binary.BigEndian.PutUint16(data[6:8], newID)

	return data
}
