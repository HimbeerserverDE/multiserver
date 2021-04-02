package main

import (
	"bytes"
	"crypto/subtle"
	"encoding/binary"
	"io"
	"log"

	"github.com/HimbeerserverDE/srp"
	"github.com/anon55555/mt/rudp"
)

const (
	ToClientHello                 = 0x02
	ToClientAuthAccept            = 0x03
	ToClientAcceptSudoMode        = 0x04
	ToClientDenySudoMode          = 0x05
	ToClientAccessDenied          = 0x0A
	ToClientBlockdata             = 0x20
	ToClientAddNode               = 0x21
	ToClientRemoveNode            = 0x22
	ToClientInventory             = 0x27
	ToClientTimeOfDay             = 0x29
	ToClientCsmRestrictionFlags   = 0x2A
	ToClientPlayerSpeed           = 0x2B
	ToClientMediaPush             = 0x2C
	ToClientChatMessage           = 0x2F
	ToClientActiveObjectRemoveAdd = 0x31
	ToClientActiveObjectMessages  = 0x32
	ToClientHp                    = 0x33
	ToClientMovePlayer            = 0x34
	ToClientFov                   = 0x36
	ToClientDeathscreen           = 0x37
	ToClientMedia                 = 0x38
	ToClientTooldef               = 0x39
	ToClientNodedef               = 0x3A
	ToClientCraftitemdef          = 0x3B
	ToClientAnnounceMedia         = 0x3C
	ToClientItemdef               = 0x3D
	ToClientPlaySound             = 0x3F
	ToClientStopSound             = 0x40
	ToClientPrivileges            = 0x41
	ToClientInventoryFormspec     = 0x42
	ToClientDetachedInventory     = 0x43
	ToClientShowFormspec          = 0x44
	ToClientMovement              = 0x45
	ToClientSpawnParticle         = 0x46
	ToClientAddParticlespawner    = 0x47
	ToClientHudAdd                = 0x49
	ToClientHudRm                 = 0x4A
	ToClientHudChange             = 0x4B
	ToClientHudSetFlags           = 0x4C
	ToClientHudSetParam           = 0x4D
	ToClientBreath                = 0x4E
	ToClientSetSky                = 0x4F
	ToClientOverrideDayNightRatio = 0x50
	ToClientLocalPlayerAnimations = 0x51
	ToClientEyeOffset             = 0x52
	ToClientDeleteParticlespawner = 0x53
	ToClientCloudParams           = 0x54
	ToClientFadeSound             = 0x55
	ToClientUpdatePlayerList      = 0x56
	ToClientModChannelMsg         = 0x57
	ToClientModChannelSignal      = 0x58
	ToClientNodeMetaChanged       = 0x59
	ToClientSetSun                = 0x5A
	ToClientSetMoon               = 0x5B
	ToClientSetStars              = 0x5C
	ToClientSrpBytesSB            = 0x60
	ToClientFormspecPrepend       = 0x61
	ToClientMinimapModes          = 0x62
)

const (
	ToServerInit            = 0x02
	ToServerInit2           = 0x11
	ToServerModChannelJoin  = 0x17
	ToServerModChannelLeave = 0x18
	ToServerModChannelMsg   = 0x19
	ToServerPlayerPos       = 0x23
	ToServerGotblocks       = 0x24
	ToServerDeletedblocks   = 0x25
	ToServerInventoryAction = 0x31
	ToServerChatMessage     = 0x32
	ToServerDamage          = 0x35
	ToServerPlayerItem      = 0x37
	ToServerRespawn         = 0x38
	ToServerInteract        = 0x39
	ToServerRemovedSounds   = 0x3A
	ToServerNodeMetaFields  = 0x3B
	ToServerInventoryFields = 0x3C
	ToServerRequestMedia    = 0x40
	ToServerClientReady     = 0x43
	ToServerFirstSrp        = 0x50
	ToServerSrpBytesA       = 0x51
	ToServerSrpBytesM       = 0x52
)

const (
	AccessDeniedWrongPassword = iota
	AccessDeniedUnexpectedData
	AccessDeniedSingleplayer
	AccessDeniedWrongVersion
	AccessDeniedWrongCharsInName
	AccessDeniedWrongName
	AccessDeniedTooManyUsers
	AccessDeniedEmptyPassword
	AccessDeniedAlreadyConnected
	AccessDeniedServerFail
	AccessDeniedCustomString
	AccessDeniedShutdown
	AccessDeniedCrash
)

func processPktCommand(src, dst *Conn, pkt *rudp.Pkt) bool {
	r := ByteReader(*pkt)

	origReader := *r
	pkt.Reader = &origReader

	cmdBytes := make([]byte, 2)
	r.Read(cmdBytes)

	if src.IsSrv() {
		switch cmd := binary.BigEndian.Uint16(cmdBytes); cmd {
		case ToClientActiveObjectRemoveAdd:
			pkt.Reader = bytes.NewReader(append(cmdBytes, processAoRmAdd(dst, r)...))
			return false
		case ToClientActiveObjectMessages:
			pkt.Reader = bytes.NewReader(append(cmdBytes, processAoMsgs(dst, r)...))
			return false
		case ToClientChatMessage:
			r.Seek(2, io.SeekCurrent)

			ReadBytes16(r)
			msg := ReadBytes16(r)

			w := bytes.NewBuffer([]byte{0x00, ToServerChatMessage})
			WriteBytes16(w, msg)

			return processServerChatMessage(dst, rudp.Pkt{
				Reader: w,
				PktInfo: rudp.PktInfo{
					Channel: pkt.Channel,
				},
			})
		case ToClientModChannelSignal:
			r.Seek(1, io.SeekCurrent)

			ch := string(ReadBytes16(r))

			state := ReadUint8(r)

			r.Seek(2, io.SeekStart)

			if ch == rpcCh {
				switch sig := ReadUint8(r); sig {
				case ModChSigJoinOk:
					src.SetUseRpc(true)
				case ModChSigSetState:
					if state == ModChStateRO {
						src.SetUseRpc(false)
					}
				}
				return true
			}

			return false
		case ToClientModChannelMsg:
			return processRpc(src, r)
		case ToClientBlockdata:
			data, drop := processBlockdata(dst, r)
			if drop {
				return true
			}

			pkt.Reader = bytes.NewReader(append(cmdBytes, data...))
			return false
		case ToClientAddNode:
			pkt.Reader = bytes.NewReader(append(cmdBytes, processAddnode(dst, r)...))
			return false
		case ToClientHudAdd:
			id := ReadUint32(r)
			dst.huds[id] = true
			return false
		case ToClientHudRm:
			id := ReadUint32(r)
			dst.huds[id] = false
			return false
		case ToClientPlaySound:
			id := int32(ReadUint32(r))
			namelen := ReadUint16(r)
			r.Seek(int64(17+namelen), io.SeekStart)
			objID := ReadUint16(r)

			if objID == dst.currentPlayerCao {
				objID = dst.localPlayerCao
			} else if objID == dst.localPlayerCao {
				objID = dst.currentPlayerCao
			}

			r.Seek(2, io.SeekStart)

			data := make([]byte, r.Len())
			r.Read(data)

			binary.BigEndian.PutUint16(data[17+namelen:19+namelen], objID)

			pkt.Reader = bytes.NewReader(append(cmdBytes, data...))

			if loop := ReadUint8(r); loop > 0 {
				dst.sounds[id] = true
			}
		case ToClientStopSound:
			id := int32(ReadUint32(r))
			dst.sounds[id] = false
		case ToClientAddParticlespawner:
			r.Seek(97, io.SeekStart)
			texturelen := ReadUint32(r)
			r.Seek(int64(6+texturelen), io.SeekCurrent)
			id := ReadUint16(r)

			if id == dst.currentPlayerCao {
				id = dst.localPlayerCao
			} else if id == dst.localPlayerCao {
				id = dst.currentPlayerCao
			}

			r.Seek(2, io.SeekStart)

			data := make([]byte, r.Len())
			r.Read(data)

			binary.BigEndian.PutUint16(data[107+texturelen:109+texturelen], id)
			pkt.Reader = bytes.NewReader(append(cmdBytes, data...))
		case ToClientInventory:
			old := *dst.Inv()

			if err := dst.Inv().Deserialize(r); err != nil {
				return true
			}

			dst.UpdateHandCapabs()

			buf := &bytes.Buffer{}
			dst.Inv().SerializeKeep(buf, old)

			pkt.Reader = bytes.NewReader(append(cmdBytes, buf.Bytes()...))

			return false
		case ToClientAccessDenied:
			doFallback, ok := ConfKey("do_fallback").(bool)
			if ok && !doFallback {
				return false
			}

			reason := ReadUint8(r)

			if reason != uint8(AccessDeniedShutdown) && reason != uint8(AccessDeniedCrash) {
				return false
			}

			msg := "shut down"
			if reason == uint8(AccessDeniedCrash) {
				msg = "crashed"
			}

			defsrv, ok := ConfKey("default_server").(string)
			if !ok {
				log.Print("Default server name not set or not a string")
				return false
			}

			if dst.ServerName() == defsrv {
				return false
			}

			dst.SendChatMsg("The minetest server has " + msg + ", connecting you to the default server...")

			go dst.Redirect(defsrv)

			for src.Forward() {
			}

			return true
		case ToClientMediaPush:
			digest := ReadBytes16(r)
			name := string(ReadBytes16(r))
			cacheByte := ReadUint8(r)
			cache := cacheByte == uint8(1)

			r.Seek(5, io.SeekCurrent)
			data := make([]byte, r.Len())
			r.Read(data)

			media[string(name)] = &mediaFile{
				digest:  digest,
				data:    data,
				noCache: !cache,
			}

			for _, conn := range Conns() {
				ack, err := conn.Send(*pkt)
				if err != nil {
					log.Print(err)
				}
				<-ack
			}

			updateMediaCache()

			return true
		default:
			return false
		}
	} else {
		switch cmd := binary.BigEndian.Uint16(cmdBytes); cmd {
		case ToServerChatMessage:
			return processChatMessage(src, *pkt)
		case ToServerFirstSrp:
			if src.sudoMode {
				src.sudoMode = false

				// This is a password change, save verifier and salt
				s := ReadBytes16(r)
				v := ReadBytes16(r)

				pwd := encodeVerifierAndSalt(s, v)

				db, err := initAuthDB()
				if err != nil {
					log.Print(err)
					return true
				}

				err = modAuthItem(db, src.Username(), pwd)
				if err != nil {
					log.Print(err)
					return true
				}

				db.Close()
			} else {
				log.Print("User " + src.Username() + " at " + src.Addr().String() + " did not enter sudo mode before attempting to change the password")
			}

			return true
		case ToServerSrpBytesA:
			if !src.sudoMode {
				A := ReadBytes16(r)

				db, err := initAuthDB()
				if err != nil {
					log.Print(err)
					return true
				}

				pwd, err := readAuthItem(db, src.Username())
				if err != nil {
					log.Print(err)
					return true
				}

				db.Close()

				s, v, err := decodeVerifierAndSalt(pwd)
				if err != nil {
					log.Print(err)
					return true
				}

				B, _, K, err := srp.Handshake(A, v)
				if err != nil {
					log.Print(err)
					return true
				}

				src.srp_s = s
				src.srp_A = A
				src.srp_B = B
				src.srp_K = K

				// Send SRP_BYTES_S_B
				data := make([]byte, 6+len(s)+len(B))
				data[0] = uint8(0x00)
				data[1] = uint8(ToClientSrpBytesSB)
				binary.BigEndian.PutUint16(data[2:4], uint16(len(s)))
				copy(data[4:4+len(s)], s)
				binary.BigEndian.PutUint16(data[4+len(s):6+len(s)], uint16(len(B)))
				copy(data[6+len(s):6+len(s)+len(B)], B)

				w := bytes.NewBuffer([]byte{0x00, ToClientSrpBytesSB})
				WriteBytes16(w, s)
				WriteBytes16(w, B)

				ack, err := src.Send(rudp.Pkt{Reader: bytes.NewReader(data)})
				if err != nil {
					log.Print(err)
					return true
				}
				<-ack
			}
			return true
		case ToServerSrpBytesM:
			if !src.sudoMode {
				M := ReadBytes16(r)
				M2 := srp.ClientProof([]byte(src.Username()), src.srp_s, src.srp_A, src.srp_B, src.srp_K)

				if subtle.ConstantTimeCompare(M, M2) == 1 {
					// Password is correct
					// Enter sudo mode
					src.sudoMode = true

					// Send ACCEPT_SUDO_MODE
					data := []byte{0, ToClientAcceptSudoMode}

					ack, err := src.Send(rudp.Pkt{Reader: bytes.NewReader(data)})
					if err != nil {
						log.Print(err)
						return true
					}
					<-ack
				} else {
					// Client supplied wrong password
					log.Print("User " + src.Username() + " at " + src.Addr().String() + " supplied wrong password for sudo mode")

					// Send DENY_SUDO_MODE
					data := []byte{0, ToClientDenySudoMode}

					ack, err := src.Send(rudp.Pkt{Reader: bytes.NewReader(data)})
					if err != nil {
						log.Print(err)
						return true
					}
					<-ack
				}
			}
			return true
		case ToServerModChannelJoin:
			deny := func() {
				data := make([]byte, 5+len(rpcCh))
				data[0] = uint8(0x00)
				data[1] = uint8(ToClientModChannelSignal)
				data[2] = uint8(ModChSigJoinFail)
				binary.BigEndian.PutUint16(data[3:5], uint16(len(rpcCh)))
				copy(data[5:], []byte(rpcCh))

				ack, err := src.Send(rudp.Pkt{Reader: bytes.NewReader(data)})
				if err != nil {
					log.Print(err)
				}
				<-ack
			}

			chAllowed, ok := ConfKey("modchannels").(bool)
			if ok && !chAllowed {
				deny()
				return true
			}

			ch := string(ReadBytes16(r))
			if ch == rpcCh {
				deny()
				return true
			}

			src.modChs[ch] = true
			return false
		case ToServerModChannelLeave:
			deny := func() {
				data := make([]byte, 5+len(rpcCh))
				data[0] = uint8(0x00)
				data[1] = uint8(ToClientModChannelSignal)
				data[2] = uint8(ModChSigLeaveFail)
				binary.BigEndian.PutUint16(data[3:5], uint16(len(rpcCh)))
				copy(data[5:], []byte(rpcCh))

				ack, err := src.Send(rudp.Pkt{Reader: bytes.NewReader(data)})
				if err != nil {
					log.Print(err)
				}
				<-ack
			}

			chAllowed, ok := ConfKey("modchannels").(bool)
			if ok && !chAllowed {
				deny()
				return true
			}

			ch := string(ReadBytes16(r))
			if ch == rpcCh {
				deny()
				return true
			}

			src.modChs[ch] = false
			return false
		case ToServerModChannelMsg:
			chAllowed, ok := ConfKey("modchannels").(bool)
			if ok && !chAllowed {
				return true
			}

			ch := string(ReadBytes16(r))
			if ch == rpcCh {
				return true
			}
			return false
		default:
			return false
		}
	}
	return false
}
