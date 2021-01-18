package multiserver

import (
	"encoding/binary"
)

func processAoRmAdd(p *Peer, data []byte) []byte {
	countRm := binary.BigEndian.Uint16(data[2:4])
	aoRm := make([]uint16, countRm)
	aoRmI := 0
	for i := uint16(0); i < countRm; i += 2 {
		aoRm[aoRmI] = binary.BigEndian.Uint16(data[4+i : 6+i])

		aoRmI++
	}

	countAdd := binary.BigEndian.Uint16(data[4+countRm*2 : 6+countRm*2])
	aoAdd := make([]uint16, countAdd)
	aoAddI := 0
	j := uint32(0)
	for i := uint32(0); i < uint32(countAdd); i++ {
		si := j + 6 + uint32(countRm)*2
		initDataLen := binary.BigEndian.Uint32(data[3+si : 7+si])

		namelen := binary.BigEndian.Uint16(data[8+si : 10+si])
		name := data[10+si : 10+si+uint32(namelen)]
		if string(name) == string(p.username) {
			if p.initAoReceived {
				binary.BigEndian.PutUint16(data[4+countRm*2:6+countRm*2], countAdd-1)
				data = append(data[:si], data[7+si+initDataLen:]...)
			} else {
				p.initAoReceived = true
			}

			j += 7 + initDataLen

			continue
		}

		aoAdd[aoAddI] = binary.BigEndian.Uint16(data[si : 2+si])

		aoAddI++

		j += 7 + initDataLen
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
