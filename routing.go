package main

import (
	"github.com/bjanders/fpanels"
)

type SwitchDevCmdRouting struct {
	trigger *fpanels.SwitchState
	cond    *fpanels.SwitchState
	cmd     DevCmd
}

type DisplayRouting struct {
	gaugeId   int
	gaugeName string
	prec      int
	freq      int
	cond      *fpanels.SwitchState
	panel     fpanels.PanelId
	display   fpanels.DisplayId
	format    string
	leds      byte
}

func (routing *SwitchDevCmdRouting) copy() *SwitchDevCmdRouting {
	var newRouting SwitchDevCmdRouting
	if routing.trigger != nil {
		newRouting.trigger = new(fpanels.SwitchState)
		*newRouting.trigger = *routing.trigger
	}
	if routing.cond != nil {
		newRouting.cond = new(fpanels.SwitchState)
		*newRouting.cond = *routing.cond
	}
	newRouting.cmd = routing.cmd
	return &newRouting
}
