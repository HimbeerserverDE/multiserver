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

		r := bytes.NewReader(buf.Bytes())
		r.Seek(1, io.SeekStart)

		count := ReadUint16(r)
		r.Seek(4, io.SeekCurrent)

	NodeLoop:
		for i := uint16(0); i < count; i++ {
			id := ReadUint16(r)
			defb := ReadBytes16(r)

			dr := bytes.NewReader(defb)
			dr.Seek(1, io.SeekStart)

			nodeName := string(ReadBytes16(dr))

			for _, srvdefs := range nodeDefs {
				for _, def := range srvdefs {
					if def.Name() == nodeName {
						nodeDefs[srv][id] = &NodeDef{id: def.ID()}
						continue NodeLoop
					}
				}
			}

			if def := NodeDefByName(srv, nodeName); def != nil {
				nodeDefs[srv][id] = &NodeDef{
					id:   def.ID(),
					name: nodeName,
					data: defb,
				}
			}

			nodeDefs[srv][id] = &NodeDef{
				id:   nextID,
				name: nodeName,
				data: defb,
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
