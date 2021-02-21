package main

import (
	"strings"

	"github.com/anon55555/mt/rudp"
)

func processInventory(p *Peer, data []byte) {
	lists := make(map[string]bool)

	inv := string(data)
	lines := strings.Split(inv, "\n")
	for _, line := range lines {
		list := strings.Split(line, " ")
		name := list[0]
		if name == "EndInventory" || name == "end" {
			return
		}
		if name == "List" {
			listname := list[1]
			lists[listname] = true
		}
		if name == "KeepList" {
			listname := list[1]
			lists[listname] = true
		}
	}

	p.invlists = lists
}

func updateHandList(p *Peer, t *ToolCapabs) error {
	item := " 1 0 " + t.String()

	list := "Width 1\n"
	list += "Item " + item + "\n"
	list += "EndInventoryList\n"

	inv := "List hand 1\n"
	inv += list
	for invlist := range p.invlists {
		inv += "KeepList " + invlist + "\n"
	}
	inv += "EndInventory\n"

	p.invlists = make(map[string]bool)

	_, err := p.Send(rudp.Pkt{Data: []byte(inv)})
	if err != nil {
		return err
	}
	return nil
}
