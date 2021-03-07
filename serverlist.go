package main

import (
	"bytes"
	"encoding/json"
	"log"
	"mime/multipart"
	"net"
	"net/http"
	"net/textproto"
	"time"
)

const (
	AnnounceStart  = "start"
	AnnounceUpdate = "update"
	AnnounceDelete = "delete"
)

func Announce(action string) error {
	listsrv, ok := GetConfKey("serverlist_url").(string)
	if !ok {
		return nil
	}

	log.Print("Updating server list announcement")

	host := GetConfKey("host").(string)

	addr, err := net.ResolveUDPAddr("udp", host)
	if err != nil {
		return err
	}

	peers := GetListener().GetPeers()

	mods, ok := GetConfKey("serverlist_mods").([]string)
	if !ok {
		mods = make([]string, 0)
	}

	clients_list := make([]string, 0)
	for _, peer := range peers {
		clients_list = append(clients_list, peer.Username())
	}

	maxPeers, ok := GetConfKey("player_limit").(int)
	if !ok {
		maxPeers = -1
	}

	conf := func(key string) interface{} {
		value, ok := GetConfKey(key).(string)
		if !ok {
			return ""
		}
		return value
	}

	confBool := func(key string) interface{} {
		value, ok := GetConfKey(key).(bool)
		if !ok {
			return false
		}
		return value
	}

	data := make(map[string]interface{})
	data["action"] = action
	data["port"] = addr.Port
	data["address"] = conf("serverlist_address")

	if action != AnnounceDelete {
		data["name"] = conf("serverlist_name")
		data["description"] = conf("serverlist_desc")
		data["version"] = "multiserver v1.8.1"
		data["proto_min"] = 37
		data["proto_max"] = 39
		data["url"] = conf("serverlist_display_url")
		data["creative"] = confBool("serverlist_creative")
		data["damage"] = confBool("serverlist_damage")
		data["password"] = confBool("disallow_empty_passwords")
		data["pvp"] = confBool("serverlist_pvp")
		data["uptime"] = Uptime()
		data["game_time"] = 0
		data["clients"] = GetPeerCount()
		data["clients_max"] = maxPeers
		data["clients_list"] = clients_list
		data["gameid"] = conf("serverlist_game")
	}

	if action == AnnounceStart {
		data["can_see_far_names"] = confBool("serverlist_can_see_far_names")
		data["mods"] = mods
	}

	s, err := json.Marshal(data)
	if err != nil {
		return err
	}

	rqBody := &bytes.Buffer{}
	writer := multipart.NewWriter(rqBody)

	mimeHeader := textproto.MIMEHeader{}
	mimeHeader.Set("Content-Disposition", "form-data; name=\"json\"")

	part, _ := writer.CreatePart(mimeHeader)
	part.Write(s)
	writer.Close()

	_, err = http.Post(listsrv+"/announce", "multipart/form-data; boundary="+writer.Boundary(), rqBody)
	if err != nil {
		return err
	}

	return nil
}

func init() {
	go func() {
		announce := time.NewTicker(5 * time.Minute)
		for {
			select {
			case <-announce.C:
				Announce(AnnounceUpdate)
			}
		}
	}()
}
