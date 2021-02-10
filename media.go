package main

import (
	"encoding/binary"
	"encoding/hex"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strings"
	"time"

	"github.com/anon55555/mt/rudp"
)

// MediaRefetchInterval is the amount of time between media downloads
const MediaRefetchInterval = 30 * time.Second

var media map[string]*mediaFile
var tooldefs [][]byte
var nodedefs map[string][]byte
var craftitemdefs [][]byte
var itemdefs [][]byte
var detachedinvs map[string][][]byte
var movement []byte
var timeofday []byte

type mediaFile struct {
	digest []byte
	data   []byte
}

func (p *Peer) fetchMedia() {
	if !p.IsSrv() {
		return
	}

	for {
		pkt, err := p.Recv()
		if err != nil {
			if err == rudp.ErrClosed {
				return
			}

			log.Print(err)
			continue
		}

		switch cmd := binary.BigEndian.Uint16(pkt.Data[0:2]); cmd {
		case ToClientTooldef:
			tooldefs = append(tooldefs, pkt.Data[2:])
		case ToClientNodedef:
			servers := GetConfKey("servers").(map[interface{}]interface{})
			var srvname string
			for server := range servers {
				if GetConfKey("servers:"+server.(string)+":address") == p.Addr().String() {
					srvname = server.(string)
				}
			}
			nodedefs[srvname] = pkt.Data[6:]
		case ToClientCraftitemdef:
			craftitemdefs = append(craftitemdefs, pkt.Data[2:])
		case ToClientItemdef:
			itemdefs = append(itemdefs, pkt.Data[2:])
		case ToClientMovement:
			movement = pkt.Data[2:]
		case ToClientDetachedInventory:
			servers := GetConfKey("servers").(map[interface{}]interface{})
			var srvname string
			for server := range servers {
				if GetConfKey("servers:"+server.(string)+":address") == p.Addr().String() {
					srvname = server.(string)
				}
			}
			detachedinvs[srvname] = append(detachedinvs[srvname], pkt.Data[2:])
		case ToClientTimeOfDay:
			timeofday = pkt.Data[2:]
		case ToClientAnnounceMedia:
			var rq []string
			count := binary.BigEndian.Uint16(pkt.Data[2:4])
			si := uint32(4)
			for i := uint16(0); i < count; i++ {
				namelen := binary.BigEndian.Uint16(pkt.Data[si : 2+si])
				name := pkt.Data[2+si : 2+si+uint32(namelen)]
				diglen := binary.BigEndian.Uint16(pkt.Data[2+si+uint32(namelen) : 4+si+uint32(namelen)])
				digest := pkt.Data[4+si+uint32(namelen) : 4+si+uint32(namelen)+uint32(diglen)]

				if media[string(name)] == nil && !isCached(string(name), digest) {
					rq = append(rq, string(name))
					media[string(name)] = &mediaFile{digest: digest}
				}

				si += 4 + uint32(namelen) + uint32(diglen)
			}

			// Request the media
			pktlen := 0
			for f := range rq {
				pktlen += 2 + len(rq[f])
			}

			data := make([]byte, 4+pktlen)
			data[0] = uint8(0x00)
			data[1] = uint8(ToServerRequestMedia)
			binary.BigEndian.PutUint16(data[2:4], uint16(len(rq)))
			sj := 4
			for f := range rq {
				binary.BigEndian.PutUint16(data[sj:2+sj], uint16(len(rq[f])))
				copy(data[2+sj:2+sj+len(rq[f])], []byte(rq[f]))
				sj += 2 + len(rq[f])
			}

			_, err := p.Send(rudp.Pkt{Data: data, ChNo: 1})
			if err != nil {
				log.Print(err)
				continue
			}
		case ToClientMedia:
			bunchcount := binary.BigEndian.Uint16(pkt.Data[2:4])
			bunch := binary.BigEndian.Uint16(pkt.Data[4:6])
			filecount := binary.BigEndian.Uint32(pkt.Data[6:10])
			si := uint32(10)
			for i := uint32(0); i < filecount; i++ {
				namelen := binary.BigEndian.Uint16(pkt.Data[si : 2+si])
				name := pkt.Data[2+si : 2+si+uint32(namelen)]
				datalen := binary.BigEndian.Uint32(pkt.Data[2+si+uint32(namelen) : 6+si+uint32(namelen)])
				data := pkt.Data[6+si+uint32(namelen) : 6+uint32(si)+uint32(namelen)+datalen]

				if media[string(name)] != nil && len(media[string(name)].data) == 0 {
					media[string(name)].data = data
				}

				si += 6 + uint32(namelen) + datalen
			}

			if bunch >= bunchcount-1 {
				p.SendDisco(0, true)
				p.Close()
				return
			}
		}
	}
}

func (p *Peer) updateDetachedInvs(srvname string) {
	for i := range detachedinvs[srvname] {
		data := make([]byte, 2+len(detachedinvs[srvname][i]))
		data[0] = uint8(0x00)
		data[1] = uint8(ToClientDetachedInventory)
		copy(data[2:], detachedinvs[srvname][i])

		ack, err := p.Send(rudp.Pkt{Data: data})
		if err != nil {
			log.Print(err)
			continue
		}
		<-ack
	}
}

func (p *Peer) announceMedia() {
	srvname, ok := GetConfKey("default_server").(string)
	if !ok {
		log.Print("Default server name not set or not a string")
		return
	}

	for _, def := range tooldefs {
		data := make([]byte, 2+len(def))
		data[0] = uint8(0x00)
		data[1] = uint8(ToClientTooldef)
		copy(data[2:], def)

		ack, err := p.Send(rudp.Pkt{Data: data})
		if err != nil {
			log.Print(err)
			continue
		}
		<-ack
	}

	data := make([]byte, 6+len(nodedef))
	data[0] = uint8(0x00)
	data[1] = uint8(ToClientNodedef)
	binary.BigEndian.PutUint32(data[2:6], uint32(len(nodedef)))
	copy(data[6:], nodedef)

	ack, err := p.Send(rudp.Pkt{Data: data})
	if err != nil {
		log.Print(err)
	}
	<-ack

	for _, def := range craftitemdefs {
		data := make([]byte, 2+len(def))
		data[0] = uint8(0x00)
		data[1] = uint8(ToClientCraftitemdef)
		copy(data[2:], def)

		ack, err := p.Send(rudp.Pkt{Data: data})
		if err != nil {
			log.Print(err)
			continue
		}
		<-ack
	}

	for _, def := range itemdefs {
		data := make([]byte, 2+len(def))
		data[0] = uint8(0x00)
		data[1] = uint8(ToClientItemdef)
		copy(data[2:], def)

		ack, err := p.Send(rudp.Pkt{Data: data})
		if err != nil {
			log.Print(err)
			continue
		}
		<-ack
	}

	p.updateDetachedInvs(srvname)

	data = make([]byte, 2+len(movement))
	data[0] = uint8(0x00)
	data[1] = uint8(ToClientMovement)
	copy(data[2:], movement)

	ack, err = p.Send(rudp.Pkt{Data: data})
	if err != nil {
		log.Print(err)
	}
	<-ack

	data = make([]byte, 2+len(timeofday))
	data[0] = uint8(0x00)
	data[1] = uint8(ToClientTimeOfDay)
	copy(data[2:], timeofday)

	ack, err = p.Send(rudp.Pkt{Data: data})
	if err != nil {
		log.Print(err)
	}
	<-ack

	csmrf, ok := GetConfKey("csm_restriction_flags").(int)
	if !ok {
		csmrf = 0
	}

	data = make([]byte, 14)
	data[0] = uint8(0x00)
	data[1] = uint8(ToClientCsmRestrictionFlags)
	binary.BigEndian.PutUint32(data[2:6], uint32(0))
	binary.BigEndian.PutUint32(data[6:10], uint32(csmrf))
	binary.BigEndian.PutUint32(data[10:], uint32(0))

	ack, err = p.Send(rudp.Pkt{Data: data})
	if err != nil {
		log.Print(err)
	}
	<-ack

	pktlen := 0
	for f := range media {
		pktlen += 4 + len(f) + len(media[f].digest)
	}

	data = make([]byte, 6+pktlen)
	data[0] = uint8(0x00)
	data[1] = uint8(ToClientAnnounceMedia)
	binary.BigEndian.PutUint16(data[2:4], uint16(len(media)))
	si := 4
	for f := range media {
		binary.BigEndian.PutUint16(data[si:2+si], uint16(len(f)))
		copy(data[2+si:2+si+len(f)], []byte(f))
		binary.BigEndian.PutUint16(data[2+si+len(f):4+si+len(f)], uint16(len(media[f].digest)))
		copy(data[4+si+len(f):4+si+len(f)+len(media[f].digest)], media[f].digest)
		si += 4 + len(f) + len(media[f].digest)
	}
	data[si] = uint8(0x00)
	data[1+si] = uint8(0x00)

	ack, err = p.Send(rudp.Pkt{Data: data})
	if err != nil {
		log.Print(err)
		return
	}
	<-ack
}

func (p *Peer) sendMedia(rqdata []byte) {
	var rq []string
	count := binary.BigEndian.Uint16(rqdata[0:2])
	si := uint16(2)
	for i := uint16(0); i < count; i++ {
		namelen := binary.BigEndian.Uint16(rqdata[si : 2+si])
		name := rqdata[2+si : 2+si+namelen]
		rq = append(rq, string(name))
		si += 2 + namelen
	}

	pktlen := 0
	for f := range rq {
		pktlen += 6 + len(rq[f]) + len(media[rq[f]].data)
	}

	data := make([]byte, 12+pktlen)
	data[0] = uint8(0x00)
	data[1] = uint8(ToClientMedia)
	data[2] = uint8(0x00)
	data[3] = uint8(0x01)
	data[4] = uint8(0x00)
	data[5] = uint8(0x00)
	binary.BigEndian.PutUint32(data[6:10], uint32(len(rq)))
	sj := 10
	for f := range rq {
		binary.BigEndian.PutUint16(data[sj:2+sj], uint16(len(rq[f])))
		copy(data[2+sj:2+sj+len(rq[f])], rq[f])
		binary.BigEndian.PutUint32(data[2+sj+len(rq[f]):6+sj+len(rq[f])], uint32(len(media[rq[f]].data)))
		copy(data[6+sj+len(rq[f]):6+sj+len(rq[f])+len(media[rq[f]].data)], media[rq[f]].data)
		sj += 6 + len(rq[f]) + len(media[rq[f]].data)
	}
	data[sj] = uint8(0x00)
	data[1+sj] = uint8(0x00)

	ack, err := p.Send(rudp.Pkt{Data: data, ChNo: 2})
	if err != nil {
		log.Print(err)
		return
	}
	<-ack
}

func loadMediaCache() error {
	os.Mkdir("cache", 0775)

	files, err := ioutil.ReadDir("cache")
	if err != nil {
		return err
	}

	for _, file := range files {
		if !file.IsDir() {
			meta := strings.Split(file.Name(), "#")
			if len(meta) != 2 {
				os.Remove("cache/" + file.Name())
				continue
			}

			data, err := ioutil.ReadFile("cache/" + file.Name())
			if err != nil {
				continue
			}

			media[meta[0]] = &mediaFile{digest: stringToDigest(meta[1]), data: data}
		}
	}

	return nil
}

func isCached(name string, digest []byte) bool {
	os.Mkdir("cache", 0775)

	_, err := os.Stat("cache/" + name + "#" + digestToString(digest))
	if os.IsNotExist(err) {
		return false
	}
	return true
}

func updateMediaCache() {
	os.Mkdir("cache", 0775)

	for mfname, mfile := range media {
		cfname := "cache/" + mfname + "#" + digestToString(mfile.digest)
		_, err := os.Stat(cfname)
		if os.IsNotExist(err) {
			ioutil.WriteFile(cfname, mfile.data, 0664)
		}
	}
}

func digestToString(d []byte) string {
	return hex.EncodeToString(d)
}

func stringToDigest(s string) []byte {
	r, err := hex.DecodeString(s)
	if err != nil {
		return []byte{}
	}
	return r
}

func loadMedia() {
	log.Print("Fetching media")

	media = make(map[string]*mediaFile)
	nodedefs = make(map[string][]byte)
	detachedinvs = make(map[string][][]byte)

	loadMediaCache()

	clt := &Peer{username: "media"}

	servers := GetConfKey("servers").(map[interface{}]interface{})
	for server := range servers {
		straddr := GetConfKey("servers:" + server.(string) + ":address")

		srvaddr, err := net.ResolveUDPAddr("udp", straddr.(string))
		if err != nil {
			log.Fatal(err)
		}

		conn, err := net.DialUDP("udp", nil, srvaddr)
		if err != nil {
			log.Fatal(err)
		}

		srv, err := Connect(conn, conn.RemoteAddr())
		if err != nil {
			log.Print(err)
			continue
		}

		fin := make(chan *Peer) // close-only
		go Init(clt, srv, false, true, fin)
		<-fin

		srv.fetchMedia()
	}

	if nodedef == nil {
		if err := mergeNodedefs(nodedefs); err != nil {
			log.Fatal(err)
		}
	}

	updateMediaCache()
}

func init() {
	loadMedia()
	go startRpc()

	go func() {
		refetch := time.NewTicker(MediaRefetchInterval)
		for {
			select {
			case <-refetch.C:
				loadMedia()
			}
		}
	}()
}
