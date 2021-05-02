package main

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"io"
	"log"
	"net"
	"os"
	"strings"

	"github.com/anon55555/mt/rudp"
)

const BytesPerBunch = 5000

var media map[string]map[string]*mediaFile
var nodedefs map[string][]byte
var itemdefs map[string][]byte
var detachedinvs map[string][][]byte

type mediaFile struct {
	digest  []byte
	data    []byte
	noCache bool
}

func PutMedia(serverName string, mediaName string, file *mediaFile) {
	if media[serverName] == nil {
		media[serverName] = make(map[string]*mediaFile)
	}
	media[serverName][mediaName] = file
}

func (c *Conn) SafeServerName() string {
	if c.Server() != nil {
		// Client has an existing connection to server, use that name
		return c.ServerName()
	}

	// Otherwise, we are certainly in client initialization, and should use the default name
	return ConfKey("default_server").(string)
}

func (c *Conn) fetchMedia() {
	if !c.IsSrv() {
		return
	}

	for {
		pkt, err := c.Recv()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return
			}

			go func() {
				<-LogReady()
				log.Print(err)
			}()
			continue
		}

		r := ByteReader(pkt)

		switch cmd := ReadUint16(r); cmd {
		case ToClientNodeDef:
			srvname := c.SafeServerName()

			r.Seek(6, io.SeekStart)

			nodedefs[srvname] = make([]byte, r.Len())
			r.Read(nodedefs[srvname])
		case ToClientItemDef:
			srvname := c.SafeServerName()

			r.Seek(6, io.SeekStart)

			itemdefs[srvname] = make([]byte, r.Len())
			r.Read(itemdefs[srvname])
		case ToClientDetachedInventory:
			srvname := c.SafeServerName()

			inv := make([]byte, r.Len())
			r.Read(inv)

			detachedinvs[srvname] = append(detachedinvs[srvname], inv)
		case ToClientAnnounceMedia:
			var rq []string

			count := ReadUint16(r)

			srvname := c.SafeServerName()

			for i := uint16(0); i < count; i++ {
				name := string(ReadBytes16(r))

				digest := ReadBytes16(r)

				if media[name] == nil && !isCached(srvname, name, digest) {
					rq = append(rq, name)
					PutMedia(srvname, name, &mediaFile{digest: digest})
				}
			}

			// Request the media
			pktlen := 0
			for f := range rq {
				pktlen += 2 + len(rq[f])
			}

			w := bytes.NewBuffer([]byte{0x00, ToServerRequestMedia})
			WriteUint16(w, uint16(len(rq)))
			for f := range rq {
				WriteBytes16(w, []byte(rq[f]))
			}

			_, err := c.Send(rudp.Pkt{
				Reader: w,
				PktInfo: rudp.PktInfo{
					Channel: 1,
				},
			})

			if err != nil {
				go func() {
					<-LogReady()
					log.Print(err)
				}()
				continue
			}
		case ToClientMedia:
			bunchCount := ReadUint16(r)
			bunchID := ReadUint16(r)
			fileCount := ReadUint32(r)

			srvname := c.SafeServerName()

			if media[srvname] == nil {
				log.Println("Attempting to store media data for un-indexed server")
				c.Close()
				return
			}

			for i := uint32(0); i < fileCount; i++ {
				name := string(ReadBytes16(r))
				data := ReadBytes32(r)

				if media[srvname][name] != nil && len(media[srvname][name].data) == 0 {
					media[srvname][name].data = data
				}
			}

			if bunchID >= bunchCount-1 {
				c.Close()
				return
			}
		}
	}
}

func (c *Conn) updateDetachedInvs(srvname string) {
	for i := range detachedinvs[srvname] {
		w := bytes.NewBuffer([]byte{0x00, ToClientDetachedInventory})
		w.Write(detachedinvs[srvname][i])

		ack, err := c.Send(rudp.Pkt{Reader: w})
		if err != nil {
			log.Print(err)
			continue
		}
		<-ack
	}
}

func (c *Conn) announceMedia() {
	// TODO Should this really be the current server name?
	srvname, ok := ConfKey("default_server").(string)
	if !ok {
		log.Print("Default server name not set or not a string")
		return
	}

	data := make([]byte, 6+len(nodedef))
	data[0] = uint8(0x00)
	data[1] = uint8(ToClientNodeDef)
	binary.BigEndian.PutUint32(data[2:6], uint32(len(nodedef)))
	copy(data[6:], nodedef)

	ack, err := c.Send(rudp.Pkt{Reader: bytes.NewReader(data)})
	if err != nil {
		log.Print(err)
	}
	<-ack

	data = make([]byte, 6+len(itemdef))
	data[0] = uint8(0x00)
	data[1] = uint8(ToClientItemDef)
	binary.BigEndian.PutUint32(data[2:6], uint32(len(itemdef)))
	copy(data[6:], itemdef)

	ack, err = c.Send(rudp.Pkt{Reader: bytes.NewReader(data)})
	if err != nil {
		log.Print(err)
	}
	<-ack

	c.updateDetachedInvs(srvname)

	csmrf, ok := ConfKey("csm_restriction_flags").(int)
	if !ok {
		csmrf = 0
	}

	csmnr, ok := ConfKey("csm_restriction_noderange").(int)
	if !ok {
		csmnr = 8
	}

	data = make([]byte, 14)
	data[0] = uint8(0x00)
	data[1] = uint8(ToClientCSMRestrictionFlags)
	binary.BigEndian.PutUint32(data[2:6], uint32(0))
	binary.BigEndian.PutUint32(data[6:10], uint32(csmrf))
	binary.BigEndian.PutUint32(data[10:], uint32(csmnr))

	ack, err = c.Send(rudp.Pkt{Reader: bytes.NewReader(data)})
	if err != nil {
		log.Print(err)
	}
	<-ack

	currentSrvname := c.SafeServerName()

	w := bytes.NewBuffer([]byte{0x00, ToClientAnnounceMedia})
	WriteUint16(w, uint16(len(media[currentSrvname])))
	for f := range media[currentSrvname] {
		WriteBytes16(w, []byte(f))
		WriteBytes16(w, media[currentSrvname][f].digest)
	}

	remote, ok := ConfKey("remote_media_server").(string)
	if !ok {
		remote = ""
	}
	WriteBytes16(w, []byte(remote))

	ack, err = c.Send(rudp.Pkt{Reader: w})
	if err != nil {
		log.Print(err)
		return
	}
	<-ack
}

func (c *Conn) sendMedia(r *bytes.Reader) {
	count := ReadUint16(r)

	var rq []string
	for i := uint16(0); i < count; i++ {
		name := string(ReadBytes16(r))
		rq = append(rq, name)
	}

	srvname := c.SafeServerName()

	bunches := []map[string]*mediaFile{make(map[string]*mediaFile)}
	var bunchlen int
	for _, f := range rq {
		bunches[len(bunches)-1][f] = media[srvname][f]
		bunchlen += len(media[srvname][f].data)

		if bunchlen >= BytesPerBunch {
			bunches = append(bunches, make(map[string]*mediaFile))
			bunchlen = 0
		}
	}

	for i, bunch := range bunches {
		w := bytes.NewBuffer([]byte{0x00, ToClientMedia})
		WriteUint16(w, uint16(len(bunches)))
		WriteUint16(w, uint16(i))
		WriteUint32(w, uint32(len(bunch)))
		for f, m := range bunch {
			WriteBytes16(w, []byte(f))
			WriteBytes32(w, m.data)
		}

		ack, err := c.Send(rudp.Pkt{
			Reader: w,
			PktInfo: rudp.PktInfo{
				Channel: 2,
			},
		})

		if err != nil {
			log.Print(err)
			return
		}
		<-ack
	}
}

func loadMediaCache() error {
	os.Mkdir("cache", 0777)

	files, err := os.ReadDir("cache")
	if err != nil {
		return err
	}

	for _, file := range files {
		if !file.IsDir() {
			meta := strings.Split(file.Name(), "#")
			if len(meta) != 3 {
				os.Remove("cache/" + file.Name())
				continue
			}

			data, err := os.ReadFile("cache/" + file.Name())
			if err != nil {
				continue
			}

			PutMedia(meta[0], meta[1], &mediaFile{digest: stringToDigest(meta[2]), data: data})
		}
	}

	return nil
}

func isCached(srvname string, name string, digest []byte) bool {
	os.Mkdir("cache", 0777)

	_, err := os.Stat("cache/" + srvname + "#" + name + "#" + digestToString(digest))
	if os.IsNotExist(err) {
		return false
	}
	return true
}

func updateMediaCache() {
	os.Mkdir("cache", 0777)

	for srvname, mcache := range media {
		for mfname, mfile := range mcache {
			if mfile.noCache {
				continue
			}

			cfname := "cache/" + srvname + "#" + mfname + "#" + digestToString(mfile.digest)
			_, err := os.Stat(cfname)
			if os.IsNotExist(err) {
				os.WriteFile(cfname, mfile.data, 0666)
			}
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

func loadMedia(servers map[string]struct{}) {
	log.Print("Fetching media")

	media = make(map[string]map[string]*mediaFile)
	detachedinvs = make(map[string][][]byte)

	loadMediaCache()

	for server := range servers {
		straddr := ConfKey("servers:" + server + ":address")

		srvaddr, err := net.ResolveUDPAddr("udp", straddr.(string))
		if err != nil {
			go func() {
				<-LogReady()
				log.Fatal(err)
			}()
		}

		conn, err := net.DialUDP("udp", nil, srvaddr)
		if err != nil {
			go func() {
				<-LogReady()
				log.Fatal(err)
			}()
		}

		srv, err := Connect(conn)
		if err != nil {
			go func() {
				<-LogReady()
				log.Print(err)
			}()
			continue
		}

		clt := &Conn{username: "media"}

		fin := make(chan *Conn) // close-only
		go Init(clt, srv, false, true, fin)
		<-fin

		srv.fetchMedia()
	}

	if err := mergeNodedefs(nodedefs); err != nil {
		go func() {
			<-LogReady()
			log.Fatal(err)
		}()
	}

	if err := mergeItemdefs(itemdefs); err != nil {
		go func() {
			<-LogReady()
			log.Fatal(err)
		}()
	}

	updateMediaCache()
}

func init() {
	nodedefs = make(map[string][]byte)
	itemdefs = make(map[string][]byte)

	servers, ok := ConfKey("servers").(map[interface{}]interface{})
	if !ok {
		go func() {
			<-LogReady()
			log.Fatal("Server list inexistent or not a dictionary")
		}()
	}

	srvs := make(map[string]struct{})
	for server := range servers {
		srvs[server.(string)] = struct{}{}
	}

	loadMedia(srvs)
}
