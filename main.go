package main

import (
	"encoding/json"
	"fmt"
	"github.com/bjanders/fpanels"
	"log"
	"net"
)

const (
	BTN_UP   = 0
	BTN_DOWN = 1
	BTN_SET  = 2
)

const (
	MOUSE_LEFT  = 1
	MOUSE_RIGHT = 2
)

const (
	CMD_AIRCRAFT  = 0
	CMD_SUBSCRIBE = 3
)

type Gauge struct {
	Id  int
	Val float64
}

var radioPanel *fpanels.RadioPanel
var multiPanel *fpanels.MultiPanel
var switchPanel *fpanels.SwitchPanel

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

var gaugeCount int

type SwitchRouting struct {
	trigger   fpanels.SwitchState
	cond      *fpanels.SwitchState
	mouse     int
	dir       int
	clickable string
	value     float64 // if dir == BTN_SET
}

var conf config

type DCS struct {
	server   string
	conn     net.Conn
	gauges   chan Gauge
	aircraft chan string
}

func (dcs *DCS) Connect(server string) (chan string, chan Gauge, error) {
	var err error
	dcs.server = server
	log.Printf("Connecting to %s", server)
	dcs.conn, err = net.Dial("tcp", server)
	if err != nil {
		log.Fatal("%v", err)
		return dcs.aircraft, dcs.gauges, err
	}
	log.Print("Connected")
	dcs.aircraft = make(chan string)
	dcs.gauges = make(chan Gauge)
	go dcs.readJSON()
	return dcs.aircraft, dcs.gauges, nil
}

func (dcs *DCS) readJSON() {
	decoder := json.NewDecoder(dcs.conn)
	var m []interface{}
	for {
		err := decoder.Decode(&m)
		if err != nil {
			log.Fatalf("!!!!!!!! %v", err)
			return
		}
		cmd := int(m[0].(float64))
		log.Printf("command %v", cmd)
		switch cmd {
		case CMD_AIRCRAFT:
			aircraft := m[1].(string)
			log.Printf("Got aircraft %s", aircraft)
			dcs.aircraft <- aircraft
		case CMD_SUBSCRIBE:
			gauges := decodeGauges(m[1:])
			for _, g := range gauges {
				dcs.gauges <- g
			}
		}
	}
}

func subscribeDisplays(dcs *DCS) {
	for _, routing := range conf.displayRouting {
		log.Printf("Subscribing to %s", routing.gaugeName)
		dcs.sendSubscribe(routing.gaugeName, routing.gaugeId)
	}
}

func decodeGauges(data []interface{}) []Gauge {
	var gauges []Gauge
	for _, g := range data {
		list := g.([]interface{})
		gauge := Gauge{int(list[0].(float64)), list[1].(float64)}
		gauges = append(gauges, gauge)
		log.Printf("gauge %v", gauge)
	}
	return gauges
}

func getPanel(panel fpanels.PanelId) {

}

func updateLEDs(routing DisplayRouting, gauge Gauge) {
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

func updateDisplay(routing DisplayRouting, gauge Gauge) {
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
			if gauge.Id == routing.gaugeId {
				if routing.panel != fpanels.RADIO && routing.leds != 0 {
					updateLEDs(routing, gauge)
				} else {
					updateDisplay(routing, gauge)
				}
			}
		}
	}
}

func routeGauge(gauge Gauge) {
	for _, routing := range conf.displayRouting {
		if gauge.Id == routing.gaugeId {
			if routing.panel != fpanels.RADIO && routing.leds != 0 {
				updateLEDs(routing, gauge)
			} else {
				updateDisplay(routing, gauge)
			}
		}
	}
}
func handleSubscribe(data []interface{}) {
	gauges := decodeGauges(data)
	log.Printf("%v", gauges)
	updateDisplays(gauges)
}

func handleAircraft(data []interface{}) {
	aircraft := data[0].(string)
	log.Printf("Got aircraft %s", aircraft)
	conf.setAircraft(aircraft)
}

func (dcs *DCS) sendSubscribe(gauge string, id int) {
	log.Print("Subscribing to ", gauge)
	fmt.Fprintf(dcs.conn, "[%d,\"%s\",{\"id\":%d}]\n", CMD_SUBSCRIBE, gauge, id)
}

func (dcs *DCS) sendClick(pos int, action int, clickable string) {
	s := fmt.Sprintf("[1,%d,%d,\"%s\"]\n", pos, action, clickable)
	//log.Print(s)
	fmt.Fprintf(dcs.conn, s)
}

func (dcs *DCS) sendClickVal(pos int, action int, clickable string, val float64) {
	fmt.Fprintf(dcs.conn, "[1,%d,%d,\"%s\",%.3f]\n", pos, action, clickable, val)
}

func handleSwitch(dcs *DCS, switchState fpanels.SwitchState) {
	for _, routing := range conf.switchRouting {
		if switchState == routing.trigger {
			if routing.dir == BTN_SET {
				dcs.sendClickVal(routing.mouse, routing.dir, routing.clickable, routing.value)
			} else {
				dcs.sendClick(routing.mouse, routing.dir, routing.clickable)
			}
		}
	}
}

func main() {
	var err error
	var radioSwitches chan fpanels.SwitchState
	var multiSwitches chan fpanels.SwitchState
	var switchSwitches chan fpanels.SwitchState
	var gauges chan Gauge
	var gauge Gauge
	var aircraftChan chan string
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
	aircraftChan, gauges, err = dcs.Connect(conf.server)
	if err != nil {
		log.Fatal(err)
	}
	log.Print("Connected")

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
	// FIX: reconnect if network connection is lost
	var switchState fpanels.SwitchState
	for {
		select {
		case switchState = <-radioSwitches:
			log.Printf("%v", switchState)
			handleSwitch(&dcs, switchState)
		case switchState = <-switchSwitches:
			log.Printf("%v", switchState)
			handleSwitch(&dcs, switchState)
		case switchState = <-multiSwitches:
			log.Printf("%v", switchState)
			handleSwitch(&dcs, switchState)
		case aircraft = <-aircraftChan:
			log.Print("Reading config")
			conf.setAircraft(aircraft)
			subscribeDisplays(&dcs)
		case gauge = <-gauges:
			routeGauge(gauge)
		}
	}
}
