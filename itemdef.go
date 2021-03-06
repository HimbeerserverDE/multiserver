package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"encoding/json"
	"io"
	"math"
	"strings"
)

var itemdef []byte
var handcapabs map[string]*ToolCapabs

type ItemDef struct {
	name string
	data []byte
}

// Name returns the name of an ItemDef
func (i *ItemDef) Name() string { return i.name }

// Data returns the actual definition
func (i *ItemDef) Data() []byte { return i.data }

type GroupCap struct {
	name     string
	uses     int16
	maxLevel int16
	times    map[int16]float32
}

// newGroupCap returns a partially initialised GroupCap
func newGroupCap(name string, uses, maxLevel int16) *GroupCap {
	return &GroupCap{
		name:     name,
		uses:     uses,
		maxLevel: maxLevel,
		times:    make(map[int16]float32),
	}
}

// Name returns the name of the group
func (g *GroupCap) Name() string { return g.name }

// Uses returns the number of uses
func (g *GroupCap) Uses() int16 { return g.uses }

// MaxLevel returns the maximum level
func (g *GroupCap) MaxLevel() int16 { return g.maxLevel }

// Times returns the digging times
func (g *GroupCap) Times() map[int16]float32 { return g.times }

// SetTimes sets the digging time for a given level
func (g *GroupCap) SetTimes(level int16, time float32) {
	g.times[level] = time
}

type ToolCapabs struct {
	fullPunchInterval float32
	maxDropLevel      int16
	groupCaps         map[string]*GroupCap
	damageGroups      map[string]int16
	punchAttackUses   uint16
}

// newToolCapabs initializes and returns ToolCapabs
func newToolCapabs(fullPunchInterval float32, maxDropLevel int16) *ToolCapabs {
	return &ToolCapabs{
		fullPunchInterval: fullPunchInterval,
		maxDropLevel:      maxDropLevel,
		groupCaps:         make(map[string]*GroupCap),
		damageGroups:      make(map[string]int16),
	}
}

// PunchInt returns the full punch interval
func (t *ToolCapabs) PunchInt() float32 { return t.fullPunchInterval }

// MaxDropLevel returns the maximum drop level
func (t *ToolCapabs) MaxDropLevel() int16 { return t.maxDropLevel }

// GroupCaps returns the group capabilities
func (t *ToolCapabs) GroupCaps() map[string]*GroupCap { return t.groupCaps }

// AddGroupCap adds a GroupCap
func (t *ToolCapabs) AddGroupCap(g *GroupCap) {
	t.groupCaps[g.Name()] = g
}

// DamageGroups returns the damage groups
func (t *ToolCapabs) DamageGroups() map[string]int16 { return t.damageGroups }

// AddDamageGroup adds a damage group
func (t *ToolCapabs) AddDamageGroup(name string, rating int16) {
	t.damageGroups[name] = rating
}

// PunchAttackUses returns the punch attack uses
func (t *ToolCapabs) PunchAttackUses() uint16 { return t.punchAttackUses }

// SetPunchAttackUses sets the punch attack uses
func (t *ToolCapabs) SetPunchAttackUses(uses uint16) {
	t.punchAttackUses = uses
}

type GroupCapJSON struct {
	MaxLevel int16     `json:"maxlevel"`
	Uses     int16     `json:"uses"`
	Times    []float32 `json:"times"`
}

type ToolCapsJSON struct {
	FullPunchInterval float32                 `json:"full_punch_interval"`
	MaxDropLevel      int16                   `json:"max_drop_level"`
	PunchAttackUses   uint16                  `json:"punch_attack_uses"`
	GroupCaps         map[string]GroupCapJSON `json:"groupcaps"`
	DamageGroups      map[string]int16        `json:"damage_groups"`
}

// SerializeJSON returns a serialized JSON string to be used in ItemMeta
func (t *ToolCapabs) SerializeJSON() (s string, err error) {
	map2array := func(m map[int16]float32) []float32 {
		var maxIndex int
		for k := range m {
			if int(k) > maxIndex {
				maxIndex = int(k)
			}
		}

		r := make([]float32, maxIndex+1)
		for k := range r {
			r[k] = -1
		}

		for k, v := range m {
			r[int(k)] = v
		}

		return r
	}

	gj := make(map[string]GroupCapJSON)
	for name, cap := range t.GroupCaps() {
		gj[name] = GroupCapJSON{
			MaxLevel: cap.MaxLevel(),
			Uses:     cap.Uses(),
			Times:    map2array(cap.Times()),
		}
	}

	tj := ToolCapsJSON{
		FullPunchInterval: t.PunchInt(),
		MaxDropLevel:      t.MaxDropLevel(),
		PunchAttackUses:   t.PunchAttackUses(),
		GroupCaps:         gj,
		DamageGroups:      t.DamageGroups(),
	}

	b, err := json.MarshalIndent(tj, "", "\t")
	if err != nil {
		return
	}

	s = strings.Replace(string(b), "-1", "null", -1)

	return
}

// DeserializeJSON processes a serialized JSON string
// and updates the ToolCapabs accordingly
func (t *ToolCapabs) DeserializeJSON(ser string) error {
	b := []byte(ser)

	tj := ToolCapsJSON{}
	if err := json.Unmarshal(b, &tj); err != nil {
		return err
	}

	r := newToolCapabs(tj.FullPunchInterval, tj.MaxDropLevel)
	r.SetPunchAttackUses(tj.PunchAttackUses)

	for name, cap := range tj.GroupCaps {
		g := newGroupCap(name, cap.Uses, cap.MaxLevel)
		for level, time := range cap.Times {
			g.SetTimes(int16(level), time)
		}
		r.AddGroupCap(g)
	}

	for g, level := range tj.DamageGroups {
		r.AddDamageGroup(g, level)
	}

	*t = *r

	return nil
}

func rmToolCapabs(def []byte) []byte {
	itemNameLen := binary.BigEndian.Uint16(def[2:4])
	desclen := binary.BigEndian.Uint16(def[4+itemNameLen : 6+itemNameLen])
	invImgLen := binary.BigEndian.Uint16(def[6+itemNameLen+desclen : 8+itemNameLen+desclen])
	wieldImgLen := binary.BigEndian.Uint16(def[8+itemNameLen+desclen+invImgLen : 10+itemNameLen+desclen+invImgLen])
	capablen := binary.BigEndian.Uint16(def[26+itemNameLen+desclen+invImgLen+wieldImgLen : 28+itemNameLen+desclen+invImgLen+wieldImgLen])

	stdcaps := []byte{
		5,
		0, 0, 0, 0,
		0, 0,
		0, 0, 0, 0,
		0, 0, 0, 0,
		0, 0,
	}

	si := 28 + itemNameLen + desclen + invImgLen + wieldImgLen

	binary.BigEndian.PutUint16(def[26+itemNameLen+desclen+invImgLen+wieldImgLen:28+itemNameLen+desclen+invImgLen+wieldImgLen], uint16(len(stdcaps)))
	return append(def[:si], append(stdcaps, def[si+capablen:]...)...)
}

func mergeItemdefs(mgrs map[string][]byte) error {
	var itemDefs []*ItemDef
	aliases := make(map[string]string)

	handcapabs = make(map[string]*ToolCapabs)
	var handDef []byte

	// Extract definitions from CItemDefManager
	for srv, compressedMgr := range mgrs {
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

			desclen := binary.BigEndian.Uint16(def[4+itemNameLen : 6+itemNameLen])
			invImgLen := binary.BigEndian.Uint16(def[6+itemNameLen+desclen : 8+itemNameLen+desclen])
			wieldImgLen := binary.BigEndian.Uint16(def[8+itemNameLen+desclen+invImgLen : 10+itemNameLen+desclen+invImgLen])
			capablen := binary.BigEndian.Uint16(def[26+itemNameLen+desclen+invImgLen+wieldImgLen : 28+itemNameLen+desclen+invImgLen+wieldImgLen])
			capab := def[28+itemNameLen+desclen+invImgLen+wieldImgLen : 28+itemNameLen+desclen+invImgLen+wieldImgLen+capablen]

			if capablen > 0 && itemName == "" {
				fpi := math.Float32frombits(binary.BigEndian.Uint32(capab[1:5]))
				mdl := int16(binary.BigEndian.Uint16(capab[5:7]))

				tcaps := newToolCapabs(fpi, mdl)

				grpCapsLen := binary.BigEndian.Uint32(capab[7:11])
				sj := uint32(11)
				for j := uint32(0); j < grpCapsLen; j++ {
					capNameLen := binary.BigEndian.Uint16(capab[sj : 2+sj])
					capName := string(capab[2+sj : 2+sj+uint32(capNameLen)])
					uses := int16(binary.BigEndian.Uint16(capab[2+sj+uint32(capNameLen) : 4+sj+uint32(capNameLen)]))
					maxlevel := int16(binary.BigEndian.Uint16(capab[4+sj+uint32(capNameLen) : 6+sj+uint32(capNameLen)]))

					gcap := newGroupCap(capName, uses, maxlevel)

					times := binary.BigEndian.Uint32(capab[6+sj+uint32(capNameLen) : 10+sj+uint32(capNameLen)])
					sk := uint32(10 + sj + uint32(capNameLen))
					for k := uint32(0); k < times; k++ {
						level := int16(binary.BigEndian.Uint16(capab[sk : 2+sk]))
						times_v := math.Float32frombits(binary.BigEndian.Uint32(capab[2+sk : 6+sk]))

						gcap.SetTimes(level, times_v)

						sk += 6
					}

					tcaps.AddGroupCap(gcap)

					sj += uint32(capNameLen) + 10 + times*6
				}

				dmgGrpCapsLen := binary.BigEndian.Uint32(capab[sj : 4+sj])
				sj += 4
				for j := uint32(0); j < dmgGrpCapsLen; j++ {
					dmgNameLen := binary.BigEndian.Uint16(capab[sj : 2+sj])
					dmgName := string(capab[2+sj : 2+sj+uint32(dmgNameLen)])
					rating := int16(binary.BigEndian.Uint16(capab[2+sj+uint32(dmgNameLen) : 4+sj+uint32(dmgNameLen)]))

					tcaps.AddDamageGroup(dmgName, rating)

					sj += 4 + uint32(dmgNameLen)
				}

				tcaps.SetPunchAttackUses(binary.BigEndian.Uint16(capab[sj : 2+sj]))

				if len(handDef) == 0 {
					handDef = def
				}
				handcapabs[srv] = tcaps

				def2 := make([]byte, len(def))
				copy(def2, def)
				binary.BigEndian.PutUint16(def2[2:4], uint16(len([]byte("multiserver:hand_"+srv))))
				def2 = append(def2[:4], append([]byte("multiserver:hand_"+srv), def2[4+itemNameLen:]...)...)

				itemDefs = append(itemDefs, &ItemDef{
					name: "multiserver:hand_" + srv,
					data: def2,
				})

				si += 2 + uint32(deflen)
				continue ItemLoop
			}

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

	if len(handDef) >= 4 {
		handdata := rmToolCapabs(handDef)

		var compHanddata bytes.Buffer
		handZw := zlib.NewWriter(&compHanddata)
		handZw.Write(handdata)
		handZw.Close()

		hand := &ItemDef{
			name: "",
			data: handdata,
		}
		itemDefs = append(itemDefs, hand)
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
