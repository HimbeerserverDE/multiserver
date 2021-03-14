package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"io"
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

		mgr := buf.Bytes()

		count := binary.BigEndian.Uint16(mgr[1:3])

		si := uint32(7)
	NodeLoop:
		for i := uint16(0); i < count; i++ {
			id := binary.BigEndian.Uint16(mgr[si : 2+si])
			deflen := binary.BigEndian.Uint16(mgr[2+si : 4+si])

			nodeNameLen := binary.BigEndian.Uint16(mgr[5+si : 7+si])
			nodeName := string(mgr[7+si : 7+si+uint32(nodeNameLen)])

			for _, srvdefs := range nodeDefs {
				for _, def := range srvdefs {
					if def.Name() == nodeName {
						nodeDefs[srv][id] = &NodeDef{id: def.ID()}
						si += 4 + uint32(deflen)
						continue NodeLoop
					}
				}
			}

			if def := NodeDefByName(srv, nodeName); def != nil {
				nodeDefs[srv][id] = &NodeDef{
					id:   def.ID(),
					name: nodeName,
					data: mgr[2+si : 4+si+uint32(deflen)],
				}
			}

			nodeDefs[srv][id] = &NodeDef{
				id:   nextID,
				name: nodeName,
				data: mgr[2+si : 4+si+uint32(deflen)],
			}

			total++

			nextID++
			if nextID == ContentUnknown {
				nextID = ContentIgnore + 1
			}

			si += 4 + uint32(deflen)
		}
	}

	// Merge definitions into new NodeDefManager
	mgr := make([]byte, 7)
	mgr[0] = uint8(1)
	binary.BigEndian.PutUint16(mgr[1:3], total)

	var allDefs []byte
	for _, srvdefs := range nodeDefs {
		for _, def := range srvdefs {
			if len(def.Data()) > 0 {
				defData := make([]byte, 2+len(def.Data()))
				binary.BigEndian.PutUint16(defData[0:2], def.ID())
				copy(defData[2:], def.Data())
				allDefs = append(allDefs, defData...)
			}
		}
	}

	binary.BigEndian.PutUint32(mgr[3:7], uint32(len(allDefs)))
	mgr = append(mgr, allDefs...)

	var compressedMgr bytes.Buffer
	zw := zlib.NewWriter(&compressedMgr)
	zw.Write(mgr)
	zw.Close()

	nodedef = compressedMgr.Bytes()

	return nil
}
