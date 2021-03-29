package main

import (
	"bytes"

	"github.com/anon55555/mt/rudp"
)

func ByteReader(pkt rudp.Pkt) *bytes.Reader {
	buf := make([]byte, rudp.MaxUnrelPktSize)
	n, _ := pkt.Read(buf)
	buf = buf[:n]

	return bytes.NewReader(buf)
}
