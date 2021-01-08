package multiserver

import "encoding/binary"

var aoIDs map[PeerID]map[uint16]bool

func InitAOMap() {
	aoIDs = make(map[PeerID]map[uint16]bool)
}

func processAORmAdd(p *Peer, data []byte) {
	countRm := binary.BigEndian.Uint16(data[2:4])
	aoRm := make([]uint16, countRm)
	aoRmI := 0
	for i := uint16(0); i < countRm; i += 2 {
		aoRm[aoRmI] = binary.BigEndian.Uint16(data[4 + i:6 + i])
		
		aoRmI++
	}
	
	countAdd := binary.BigEndian.Uint16(data[4 + countRm * 2:6 + countRm * 2])
	aoAdd := make([]uint16, countAdd)
	aoAddI := 0
	j := uint32(0)
	for i := uint32(0); i < uint32(countAdd); i++ {
		si := j + 6 + uint32(countRm) * 2
		initDataLen := binary.BigEndian.Uint32(data[3 + si:7 + si])
		
		if data[2 + si] == uint8(0x65) && !p.initAoReceived {
			p.initAoReceived = true
			j += 7 + initDataLen
			continue
		}
		
		aoAdd[aoAddI] = binary.BigEndian.Uint16(data[si:2 + si])
		
		aoAddI++
		
		j += 7 + initDataLen
	}
	
	for i := range aoAdd {
		if aoAdd[i] != 0 {
			aoIDs[p.ID()][aoAdd[i]] = true
		}
	}
	
	for i := range aoRm {
		aoIDs[p.ID()][aoRm[i]] = false
	}
}
