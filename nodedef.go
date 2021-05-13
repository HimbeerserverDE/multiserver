package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"io"
	"log"
	"regexp"
	"sync"
)

var nodedef []byte

const (
	ContentUnknown = 125
	ContentAir     = 126
	ContentIgnore  = 127
)

type NodeDef struct {
	id   uint16
	name string
	data []byte
}

var nodeDefsMu sync.RWMutex
var nodeDefs map[string]map[uint16]*NodeDef

// NodeDefByName returns the NodeDef that has the specified name on a
// minetest server
func NodeDefByName(srv, name string) *NodeDef {
	for _, def := range nodeDefs[srv] {
		if def.Name() == name {
			return def
		}
	}

	return nil
}

// ID returns the content ID of a NodeDef
func (n *NodeDef) ID() uint16 {
	if n == nil {
		return ContentUnknown
	}
	return n.id
}

// Name returns the name of a NodeDef
func (n *NodeDef) Name() string {
	if n == nil {
		return ""
	}
	return n.name
}

// Data returns the actual definition
func (n *NodeDef) Data() []byte {
	if n == nil {
		return []byte{}
	}
	return n.data
}

// NodeDefs returns all node definitions
func NodeDefs() map[string]map[uint16]*NodeDef {
	nodeDefsMu.RLock()
	defer nodeDefsMu.RUnlock()

	return nodeDefs
}

const TAT_VERTICAL_FRAMES uint8 = 1
const TAT_SHEET_2D uint8 = 2

const TILE_FLAG_HAS_COLOR uint16 = 1 << 3
const TILE_FLAG_HAS_SCALE uint16 = 1 << 4
const TILE_FLAG_HAS_ALIGN_STYLE uint16 = 1 << 5

var reTexture = regexp.MustCompile(`(?:[^\^(]*)[.]png`)

func processTileDef(srv string, dw io.Writer, dr io.Reader) {
	WriteUint8(dw, ReadUint8(dr))

	// Reference unique media path
	name := ReadBytes16(dr)
	processed := reTexture.ReplaceAllFunc(name, func(match []byte) []byte {
		return []byte(srv + "__" + string(match))
	})
	WriteBytes16(dw, processed)

	animType := ReadUint8(dr)
	WriteUint8(dw, animType)

	animLen := 0
	if animType == TAT_VERTICAL_FRAMES {
		animLen = 8
	} else if animType == TAT_SHEET_2D {
		animLen = 6
	}
	anim := make([]byte, animLen)
	io.ReadFull(dr, anim)
	dw.Write(anim)

	flags := ReadUint16(dr)
	WriteUint16(dw, flags)

	restLen := 0
	if flags&TILE_FLAG_HAS_COLOR > 0 {
		restLen += 3
	}

	if flags&TILE_FLAG_HAS_SCALE > 0 {
		restLen++
	}

	if flags&TILE_FLAG_HAS_ALIGN_STYLE > 0 {
		restLen++
	}
	rest := make([]byte, restLen)
	io.ReadFull(dr, rest)
	dw.Write(rest)
}

func mergeNodedefs(mgrs map[string][]byte) error {
	var total uint16

	nodeDefsMu.Lock()
	defer nodeDefsMu.Unlock()

	if nodeDefs == nil {
		nodeDefs = make(map[string]map[uint16]*NodeDef)
	}

	var nextID uint16

	// Extract definitions from NodeDefManagers
	for srv, compressedMgr := range mgrs {
		if nodeDefs[srv] == nil {
			nodeDefs[srv] = make(map[uint16]*NodeDef)
		}

		zr, err := zlib.NewReader(bytes.NewReader(compressedMgr))
		if err != nil {
			return err
		}

		buf := &bytes.Buffer{}
		_, err = io.Copy(buf, zr)
		if err != nil {
			return err
		}
		zr.Close()

		r := bytes.NewReader(buf.Bytes())
		r.Seek(1, io.SeekStart)

		count := ReadUint16(r)
		r.Seek(4, io.SeekCurrent)

	NodeLoop:
		for i := uint16(0); i < count; i++ {
			id := ReadUint16(r)
			defb := ReadBytes16(r)

			dw := bytes.NewBuffer(make([]byte, 0))
			// ContentFeatures
			dr := bytes.NewReader(defb)
			// Version
			WriteUint8(dw, ReadUint8(dr))

			nodeName := string(ReadBytes16(dr))
			WriteBytes16(dw, []byte(nodeName))

			// Groups
			groupCount := ReadUint16(dr)
			WriteUint16(dw, groupCount)
			for g := uint16(0); g < groupCount; g++ {
				WriteBytes16(dw, ReadBytes16(dr))
				WriteUint16(dw, ReadUint16(dr))
			}

			// Param1 and Param2
			WriteUint8(dw, ReadUint8(dr))
			WriteUint8(dw, ReadUint8(dr))

			// DrawType
			WriteUint8(dw, ReadUint8(dr))
			// Mesh
			WriteBytes16(dw, ReadBytes16(dr))
			// Scale (float)
			WriteUint32(dw, ReadUint32(dr))

			// Tiledef count (always 6)
			WriteUint8(dw, ReadUint8(dr))
			for t := 0; t < 6; t++ {
				processTileDef(srv, dw, dr)
			}
			for t := 0; t < 6; t++ {
				processTileDef(srv, dw, dr)
			}
			specialCount := ReadUint8(dr)
			WriteUint8(dw, specialCount)
			for t := uint8(0); t < specialCount; t++ {
				processTileDef(srv, dw, dr)
			}
			rest, err := io.ReadAll(dr)
			if err != nil {
				log.Fatal(err)
			}
			dw.Write(rest)

			for _, srvdefs := range nodeDefs {
				for _, def := range srvdefs {
					if def.Name() == nodeName {
						nodeDefs[srv][id] = &NodeDef{id: def.ID()}
						continue NodeLoop
					}
				}
			}

			defu := dw.Bytes()

			nodeDefs[srv][id] = &NodeDef{
				id:   nextID,
				name: nodeName,
				data: defu,
			}

			total++

			nextID++
			if nextID == ContentUnknown {
				nextID = ContentIgnore + 1
			}
		}
	}

	// Merge definitions into new NodeDefManager
	mgr := &bytes.Buffer{}
	mgr.WriteByte(1)
	WriteUint16(mgr, total)

	var allDefs []byte
	for _, srvdefs := range nodeDefs {
		for _, def := range srvdefs {
			if len(def.Data()) > 0 {
				defData := make([]byte, 4+len(def.Data()))
				binary.BigEndian.PutUint16(defData[0:2], def.ID())
				binary.BigEndian.PutUint16(defData[2:4], uint16(len(def.Data())))
				copy(defData[4:], def.Data())
				allDefs = append(allDefs, defData...)
			}
		}
	}

	WriteBytes32(mgr, allDefs)

	var compressedMgr bytes.Buffer
	zw := zlib.NewWriter(&compressedMgr)
	zw.Write(mgr.Bytes())
	zw.Close()

	nodedef = compressedMgr.Bytes()

	return nil
}
