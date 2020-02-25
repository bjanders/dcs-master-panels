package main

import (
	"fmt"
	"log"
	"time"

	"github.com/bjanders/fpanels"
)

var radioPanel *fpanels.RadioPanel
var multiPanel *fpanels.MultiPanel
var switchPanel *fpanels.SwitchPanel

var gaugeCount int

var conf config

func subscribeDisplays(dcs *DCS) {
	for _, routing := range conf.displayRouting {
		log.Printf("Subscribing to %s", routing.gaugeName)
		dcs.sendSubscribe(routing.gaugeName, routing.gaugeID, routing.prec)
	}
}

func updateLEDs(routing *DisplayRouting, gauge Gauge) {
	var panel fpanels.LEDDisplayer
	switch routing.panel {
	case fpanels.SWITCH:
		panel = switchPanel
	case fpanels.MULTI:
		panel = multiPanel
	default:
		return
	}
	if panel != nil {
		panel.LEDsOnOff(routing.leds, gauge.Val)
	}
}

func updateDisplay(routing *DisplayRouting, gauge Gauge) {
	var panel fpanels.StringDisplayer

	switch routing.panel {
	case fpanels.RADIO:
		panel = radioPanel
	case fpanels.MULTI:
		panel = multiPanel
	default:
		return
	}
	if panel != nil {
		s := fmt.Sprintf(routing.format, gauge.Val)
		panel.DisplayString(routing.display, s)
	}
}

func updateDisplays(gauges []Gauge) {
	for _, gauge := range gauges {
		for _, routing := range conf.displayRouting {
			if gauge.Id == routing.gaugeID {
				if routing.panel != fpanels.RADIO && routing.leds != 0 {
					updateLEDs(routing, gauge)
				} else {
					updateDisplay(routing, gauge)
				}
			}
		}
	}
}

func checkCond(cond *fpanels.SwitchState) bool {
	if cond == nil {
		return true
	}
	switch cond.Panel {
	case fpanels.RADIO:
		return cond.On == radioPanel.IsSwitchSet(cond.Switch)
	case fpanels.MULTI:
		return cond.On == multiPanel.IsSwitchSet(cond.Switch)
	case fpanels.SWITCH:
		return cond.On == switchPanel.IsSwitchSet(cond.Switch)
	}
	return true
}

func routeGauge(gauge Gauge) {
	// FIX: Save all gauges so that displays can be updated
	//      if state changes
	for _, routing := range conf.displayRouting {
		if gauge.Id == routing.gaugeID && checkCond(routing.cond) {
			if routing.panel != fpanels.RADIO && routing.leds != 0 {
				updateLEDs(routing, gauge)
			} else {
				updateDisplay(routing, gauge)
			}
		}
	}
}

func handleSwitch(dcs *DCS, switchState *fpanels.SwitchState) {
	// FIX: Update displays since the state may have changed
	for _, devRouting := range conf.devCmdRouting {
		if *switchState == *devRouting.trigger && checkCond(devRouting.cond) {
			dcs.sendDevCmd(&devRouting.cmd)
		}
	}
}

func main() {
	var err error
	var radioSwitches chan fpanels.SwitchState
	var multiSwitches chan fpanels.SwitchState
	var switchSwitches chan fpanels.SwitchState
	var gauge Gauge
	var aircraft string
	var dcs DCS

	conf.getServer()

	radioPanel, err = fpanels.NewRadioPanel()
	if err != nil {
		log.Print("%v", err)
	}
	switchPanel, err = fpanels.NewSwitchPanel()
	if err != nil {
		log.Print("%v", err)
	}
	multiPanel, err = fpanels.NewMultiPanel()
	if err != nil {
		log.Print("%v", err)
	}

	if radioPanel != nil {
		radioSwitches = radioPanel.WatchSwitches()
	}
	if switchPanel != nil {
		switchSwitches = switchPanel.WatchSwitches()
	}
	if multiPanel != nil {
		multiSwitches = multiPanel.WatchSwitches()
	}
	// FIX: preiodically try to connect to unconnected panels
	for {
		connected := false
		log.Printf("Connecting to DCS at %s", conf.server)
		err = dcs.Connect(conf.server)
		if err != nil {
			log.Print(err)
			log.Print("Waiting 10 seconds")
			time.Sleep(10 * time.Second)
		} else {
			log.Print("Connected")
			connected = true
		}

		var switchState fpanels.SwitchState
		for connected {
			select {
			case switchState = <-radioSwitches:
				handleSwitch(&dcs, &switchState)
			case switchState = <-switchSwitches:
				handleSwitch(&dcs, &switchState)
			case switchState = <-multiSwitches:
				handleSwitch(&dcs, &switchState)
			case aircraft = <-dcs.Ch.aircraft:
				log.Printf("Controlling aircraft %s", aircraft)
				conf.setAircraft(aircraft)
				subscribeDisplays(&dcs)
			case gauge = <-dcs.Ch.gauges:
				routeGauge(gauge)
			case connected = <-dcs.Ch.connected:
				log.Print("Lost connection")
			}
		}
	}
}
