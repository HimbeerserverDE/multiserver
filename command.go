package main

import (
	"bytes"
	"crypto/subtle"
	"encoding/binary"
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

func processPktCommand(src, dst *Peer, pkt *rudp.Pkt) bool {
	if src.IsSrv() {
		switch cmd := binary.BigEndian.Uint16(pkt.Data[0:2]); cmd {
		case ToClientActiveObjectRemoveAdd:
			pkt.Data = processAoRmAdd(dst, pkt.Data)
			return false
		case ToClientActiveObjectMessages:
			pkt.Data = processAoMsgs(dst, pkt.Data)
			return false
		case ToClientChatMessage:
			namelen := binary.BigEndian.Uint16(pkt.Data[4:6])
			msglen := binary.BigEndian.Uint16(pkt.Data[6+namelen : 8+namelen])
			msg := pkt.Data[8+namelen:]

			data := make([]byte, 4+msglen*2)
			data[0] = uint8(0x00)
			data[1] = uint8(ToServerChatMessage)
			binary.BigEndian.PutUint16(data[2:4], uint16(msglen))
			copy(data[4:], msg)

			return processServerChatMessage(dst, rudp.Pkt{Data: data, ChNo: pkt.ChNo})
		case ToClientModChannelSignal:
			chlen := binary.BigEndian.Uint16(pkt.Data[3:5])
			ch := string(pkt.Data[5 : 5+chlen])
			if ch == rpcCh {
				switch sig := pkt.Data[2]; sig {
				case ModChSigJoinOk:
					src.SetUseRpc(true)
				case ModChSigSetState:
					state := pkt.Data[5+chlen]
					if state == ModChStateRO {
						src.SetUseRpc(false)
					}
				}
				return true
			}
			return false
		case ToClientModChannelMsg:
			return processRpc(src, *pkt)
		case ToClientBlockdata:
			return processBlockdata(dst, pkt)
		case ToClientAddNode:
			return processAddnode(dst, pkt)
		case ToClientHudAdd:
			id := binary.BigEndian.Uint32(pkt.Data[2:6])
			dst.huds[id] = true
			return false
		case ToClientHudRm:
			id := binary.BigEndian.Uint32(pkt.Data[2:6])
			dst.huds[id] = false
			return false
		case ToClientPlaySound:
			id := int32(binary.BigEndian.Uint32(pkt.Data[2:6]))
			namelen := binary.BigEndian.Uint16(pkt.Data[6:8])
			objID := binary.BigEndian.Uint16(pkt.Data[17+namelen : 19+namelen])
			if objID == dst.currentPlayerCao {
				objID = dst.localPlayerCao
			} else if objID == dst.localPlayerCao {
				objID = dst.currentPlayerCao
			}
			binary.BigEndian.PutUint16(pkt.Data[17+namelen:19+namelen], objID)
			if loop := pkt.Data[19+namelen]; loop > 0 {
				dst.sounds[id] = true
			}
		case ToClientStopSound:
			id := int32(binary.BigEndian.Uint32(pkt.Data[2:6]))
			dst.sounds[id] = false
		case ToClientAddParticlespawner:
			texturelen := binary.BigEndian.Uint32(pkt.Data[97:101])
			id := binary.BigEndian.Uint16(pkt.Data[107+texturelen : 109+texturelen])
			if id == dst.currentPlayerCao {
				id = dst.localPlayerCao
			} else if id == dst.localPlayerCao {
				id = dst.currentPlayerCao
			}
			binary.BigEndian.PutUint16(pkt.Data[107+texturelen:109+texturelen], id)
		case ToClientInventory:
			if err := dst.Inv().Deserialize(bytes.NewReader(pkt.Data[2:])); err != nil {
				return true
			}

			dst.UpdateHandCapabs()

			buf := &bytes.Buffer{}
			dst.Inv().Serialize(buf)

			pkt.Data = append(pkt.Data[:2], buf.Bytes()...)

			return false
		case ToClientAccessDenied:
			doFallback, ok := ConfKey("do_fallback").(bool)
			if ok && !doFallback {
				return false
			}

			if pkt.Data[2] != uint8(11) && pkt.Data[2] != uint8(12) {
				return false
			}

			msg := "shut down"
			if pkt.Data[2] == uint8(12) {
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
			digLen := binary.BigEndian.Uint16(pkt.Data[2:4])
			digest := pkt.Data[4 : 4+digLen]
			namelen := binary.BigEndian.Uint16(pkt.Data[4+digLen : 6+digLen])
			name := pkt.Data[6+digLen : 6+digLen+namelen]
			cache := pkt.Data[6+digLen+namelen] == uint8(1)
			data := pkt.Data[11+digLen+namelen:]

			media[string(name)] = &mediaFile{
				digest:  digest,
				data:    data,
				noCache: !cache,
			}

			for _, peer := range Peers() {
				ack, err := peer.Send(*pkt)
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
		switch cmd := binary.BigEndian.Uint16(pkt.Data[0:2]); cmd {
		case ToServerChatMessage:
			return processChatMessage(src, *pkt)
		case ToServerFirstSrp:
			if src.sudoMode {
				src.sudoMode = false

				// This is a password change, save verifier and salt
				lenS := binary.BigEndian.Uint16(pkt.Data[2:4])
				s := pkt.Data[4 : 4+lenS]

				lenV := binary.BigEndian.Uint16(pkt.Data[4+lenS : 6+lenS])
				v := pkt.Data[6+lenS : 6+lenS+lenV]

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
				lenA := binary.BigEndian.Uint16(pkt.Data[2:4])
				A := pkt.Data[4 : 4+lenA]

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

				ack, err := src.Send(rudp.Pkt{Data: data})
				if err != nil {
					log.Print(err)
					return true
				}
				<-ack
			}
			return true
		case ToServerSrpBytesM:
			if !src.sudoMode {
				lenM := binary.BigEndian.Uint16(pkt.Data[2:4])
				M := pkt.Data[4 : 4+lenM]

				M2 := srp.CalculateM([]byte(src.Username()), src.srp_s, src.srp_A, src.srp_B, src.srp_K)

				if subtle.ConstantTimeCompare(M, M2) == 1 {
					// Password is correct
					// Enter sudo mode
					src.sudoMode = true

					// Send ACCEPT_SUDO_MODE
					data := []byte{0, ToClientAcceptSudoMode}

					ack, err := src.Send(rudp.Pkt{Data: data})
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

					ack, err := src.Send(rudp.Pkt{Data: data})
					if err != nil {
						log.Print(err)
						return true
					}
					<-ack
				}
			}
			return true
		case ToServerModChannelJoin:
			ch := string(pkt.Data[4:])
			if ch == rpcCh {
				data := make([]byte, 5+len(rpcCh))
				data[0] = uint8(0x00)
				data[1] = uint8(ToClientModChannelSignal)
				data[2] = uint8(ModChSigJoinFail)
				binary.BigEndian.PutUint16(data[3:5], uint16(len(rpcCh)))
				copy(data[5:], []byte(rpcCh))

				ack, err := src.Send(rudp.Pkt{Data: data})
				if err != nil {
					log.Print(err)
				}
				<-ack

				return true
			}

			src.modChs[ch] = true
			return false
		case ToServerModChannelLeave:
			ch := string(pkt.Data[4:])
			if ch == rpcCh {
				data := make([]byte, 5+len(rpcCh))
				data[0] = uint8(0x00)
				data[1] = uint8(ToClientModChannelSignal)
				data[2] = uint8(ModChSigLeaveFail)
				binary.BigEndian.PutUint16(data[3:5], uint16(len(rpcCh)))
				copy(data[5:], []byte(rpcCh))

				ack, err := src.Send(rudp.Pkt{Data: data})
				if err != nil {
					log.Print(err)
				}
				<-ack

				return true
			}

			src.modChs[ch] = false
			return false
		case ToServerModChannelMsg:
			chlen := binary.BigEndian.Uint16(pkt.Data[2:4])
			ch := string(pkt.Data[4 : 4+chlen])
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
