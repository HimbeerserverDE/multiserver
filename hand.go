package main

import (
	"github.com/anon55555/mt"
)

func (p *Peer) UpdateHandCapabs() error {
	l := p.Inv().List("hand")
	if l == nil {
		*p.inv = mt.Inv(append([]mt.NamedInvList(*p.inv), mt.NamedInvList{
			Name: "hand",
			InvList: mt.InvList{
				Width: 1,
			},
		}))
		l = p.Inv().List("hand")
	}

	var hand mt.Stack

	if len(l.Stacks) == 1 && l.Stacks[0].Name != "multiserver:hand_"+p.ServerName() {
		hand = l.Stacks[0]

		s, err := handcapabs[p.ServerName()].SerializeJSON()
		if err != nil {
			return err
		}

		hand.SetField("tool_capabilities", s)
	} else {
		hand = mt.Stack{
			Item: mt.Item{
				Name: "multiserver:hand_" + p.ServerName(),
			},
			Count: 1,
		}
	}

	l.Stacks = []mt.Stack{hand}

	return nil
}
