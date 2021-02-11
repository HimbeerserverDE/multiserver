package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"io"
)

var itemdef []byte

type ItemDef struct {
	name string
	data []byte
}

// Name returns the name of an ItemDef
func (i *ItemDef) Name() string { return i.name }

// Data returns the actual definition
func (i *ItemDef) Data() []byte { return i.data }

func mergeItemdefs(mgrs [][]byte) error {
	var itemDefs []*ItemDef
	aliases := make(map[string]string)

	// Extract definitions from CItemDefManager
	for _, compressedMgr := range mgrs {
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

		si := uint32(3)
	ItemLoop:
		for i := uint16(0); i < count; i++ {
			deflen := binary.BigEndian.Uint16(mgr[si : 2+si])
			def := mgr[2+si : 2+si+uint32(deflen)]

			itemNameLen := binary.BigEndian.Uint16(def[2:4])
			itemName := string(def[4 : 4+itemNameLen])

			for _, idef := range itemDefs {
				if idef.Name() == itemName {
					si += 2 + uint32(deflen)
					continue ItemLoop
				}
			}

			itemDefs = append(itemDefs, &ItemDef{name: itemName, data: def})

			si += 2 + uint32(deflen)
		}

		aliasCount := binary.BigEndian.Uint16(mgr[si : 2+si])

		si += 2
		for i := uint16(0); i < aliasCount; i++ {
			namelen := binary.BigEndian.Uint16(mgr[si : 2+si])
			name := string(mgr[2+si : 2+si+uint32(namelen)])

			convertlen := binary.BigEndian.Uint16(mgr[2+si+uint32(namelen) : 4+si+uint32(namelen)])
			convert := string(mgr[4+si+uint32(namelen) : 4+si+uint32(namelen)+uint32(convertlen)])

			if aliases[name] == "" {
				aliases[name] = convert
			}

			si += 4 + uint32(namelen) + uint32(convertlen)
		}
	}

	// Merge definitions into new CItemDefManager
	mgr := make([]byte, 3)
	mgr[0] = uint8(0x00)
	binary.BigEndian.PutUint16(mgr[1:3], uint16(len(itemDefs)))

	var allDefs []byte
	for _, def := range itemDefs {
		defData := make([]byte, 2+len(def.Data()))
		binary.BigEndian.PutUint16(defData[0:2], uint16(len(def.Data())))
		copy(defData[2:], def.Data())
		allDefs = append(allDefs, defData...)
	}

	mgr = append(mgr, allDefs...)

	aliasCount := make([]byte, 2)
	binary.BigEndian.PutUint16(aliasCount, uint16(len(aliases)))
	mgr = append(mgr, aliasCount...)

	for name, convert := range aliases {
		namelen := make([]byte, 2)
		binary.BigEndian.PutUint16(namelen, uint16(len(name)))

		convertlen := make([]byte, 2)
		binary.BigEndian.PutUint16(convertlen, uint16(len(convert)))

		mgr = append(mgr, namelen...)
		mgr = append(mgr, []byte(name)...)

		mgr = append(mgr, convertlen...)
		mgr = append(mgr, []byte(convert)...)
	}

	var compressedMgr bytes.Buffer
	zw := zlib.NewWriter(&compressedMgr)
	zw.Write(mgr)
	zw.Close()

	itemdef = compressedMgr.Bytes()

	return nil
}
