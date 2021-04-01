package main

import (
	"bytes"
	"encoding/binary"
	"io"

	"github.com/anon55555/mt/rudp"
)

func ByteReader(pkt rudp.Pkt) *bytes.Reader {
	buf := make([]byte, rudp.MaxUnrelPktSize)
	n, _ := pkt.Read(buf)
	buf = buf[:n]

	return bytes.NewReader(buf)
}

func ReadUint8(r io.Reader) uint8 {
	b := make([]byte, 1)
	r.Read(b)
	return uint8(b[0])
}

func WriteUint8(w io.Writer, v uint8) {
	w.Write([]byte{v})
}

func ReadUint16(r io.Reader) uint16 {
	b := make([]byte, 2)
	r.Read(b)
	return binary.BigEndian.Uint16(b)
}

func WriteUint16(w io.Writer, v uint16) {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, v)
	w.Write(b)
}

func ReadUint32(r io.Reader) uint32 {
	b := make([]byte, 4)
	r.Read(b)
	return binary.BigEndian.Uint32(b)
}

func WriteUint32(w io.Writer, v uint32) {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, v)
	w.Write(b)
}

func ReadUint64(r io.Reader) uint64 {
	b := make([]byte, 8)
	r.Read(b)
	return binary.BigEndian.Uint64(b)
}

func WriteUint64(w io.Writer, v uint64) {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, v)
	w.Write(b)
}
