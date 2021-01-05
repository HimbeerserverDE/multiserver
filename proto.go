package multiserver

// protoID must be at the start of every network packet
const protoID uint32 = 0x4f457403

// PeerIDs aren't actually used to identify peers, network addresses
// are, these just exist for backwards compatibility
type PeerID uint16

const (
	// Used by clients before the server sets their ID
	PeerIDNil PeerID = iota
	
	// The server always has this ID
	PeerIDSrv
	
	// Lowest ID the server can assign to a client
	PeerIDCltMin
)

// ChannelCount is the maximum channel number + 1
const ChannelCount = 3

type rawPkt struct {
	Data	[]byte
	ChNo	uint8
	Unrel	bool
}

type rawType uint8

const (
	rawTypeCtl rawType = iota
	rawTypeOrig
	rawTypeSplit
	rawTypeRel
)

type ctlType uint8

const (
	ctlAck ctlType = iota
	ctlSetPeerID
	ctlPing
	ctlDisco
)

type Pkt struct {
	Data	[]byte
	ChNo	uint8
	Unrel	bool
}

// seqnums are sequence numbers used to maintain reliable packet order
// and to identify split packets
type seqnum uint16

const seqnumInit seqnum = 65500
