package main

import (
	"fmt"

	"github.com/anon55555/mt"
)

func (c *Conn) UpdateHandCapabs() error {
	l := c.Inv().List("hand")
	if l == nil {
		*c.inv = mt.Inv(append([]mt.NamedInvList(*c.inv), mt.NamedInvList{
			Name: "hand",
			InvList: mt.InvList{
				Width: 1,
			},
		}))
		l = c.Inv().List("hand")
	}

	var hand mt.Stack

	if len(l.Stacks) == 1 && l.Stacks[0].Name != "multiserver:hand_"+c.ServerName() {
		hand = l.Stacks[0]

		caps := handcapabs[c.ServerName()]
		if caps == nil {
			return fmt.Errorf("hand tool capabilities of server %s missing", c.ServerName())
		}

		s, err := caps.SerializeJSON()
		if err != nil {
			return err
		}

		hand.SetField("tool_capabilities", s)
	} else {
		hand = mt.Stack{
			Item: mt.Item{
				Name: "multiserver:hand_" + c.ServerName(),
			},
			Count: 1,
		}
	}

	l.Stacks = []mt.Stack{hand}

	return nil
}
