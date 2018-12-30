package main

import (
	"encoding/json"
	"fmt"
	"github.com/bjanders/fpanels"
	"log"
	"net"
)

//const server = "127.0.0.1:8888"
const server = "192.168.71.153:8888"

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

type Display int

const (
	RADIO_1 Display = iota
	RADIO_2
	RADIO_3
	RADIO_4
	MULTI_1
	MULTI_2
	GEAR_N_RED
	GEAR_N_GREEN
	GEAR_L_RED
	GEAR_L_GREEN
	GEAR_R_RED
	GEAR_R_GREEN
)

var radioPanel *fpanels.RadioPanel
var multiPanel *fpanels.MultiPanel
var switchPanel *fpanels.SwitchPanel

type DisplayRouting struct {
	gaugeId   int
	gaugeName string
	cond      fpanels.SwitchState
	panel     fpanels.PanelId
	display   fpanels.DisplayId
	format    string
}

var gaugeCount int

type SwitchRouting struct {
	trigger   fpanels.SwitchState
	cond      fpanels.SwitchState
	mouse     int
	dir       int
	clickable string
	value     float64
}

var conf config

func SubscribeDisplay(conn net.Conn, gaugeName string, panel fpanels.PanelId, display fpanels.DisplayId, format string) {
	routing := DisplayRouting{
		gaugeCount,
		gaugeName,
		fpanels.SwitchState{},
		panel,
		display,
		format,
	}
	conf.displayRouting = append(conf.displayRouting, routing)
	sendSubscribe(conn, routing.gaugeName, routing.gaugeId)
	gaugeCount++
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

func updateDisplays(gauges []Gauge) {
	for _, gauge := range gauges {
		for _, routing := range conf.displayRouting {
			if gauge.Id == routing.gaugeId {
				switch routing.panel {
				case fpanels.RADIO:
					s := fmt.Sprintf(routing.format, gauge.Val)
					radioPanel.DisplayString(routing.display, s)
				}
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
	conf.aircraft = data[0].(string)
	log.Printf("Got aircraft %s", conf.aircraft)
	conf.read()
}

func readJSON(conn net.Conn) {
	decoder := json.NewDecoder(conn)
	var m []interface{}
	for {
		err := decoder.Decode(&m)
		if err != nil {
			log.Fatalf("%v", err)
			return
		}
		cmd := int(m[0].(float64))
		log.Printf("command %v", cmd)
		switch cmd {
		case CMD_AIRCRAFT:
			handleAircraft(m[1:])
		case CMD_SUBSCRIBE:
			handleSubscribe(m[1:])
		}
	}
}

func sendSubscribe(conn net.Conn, gauge string, id int) {
	fmt.Fprintf(conn, "[%d,\"%s\",{\"id\":%d}]\n", CMD_SUBSCRIBE, gauge, id)
}

func sendClick(conn net.Conn, pos int, action int, clickable string) {
	s := fmt.Sprintf("[1,%d,%d,\"%s\"]\n", pos, action, clickable)
	log.Print(s)
	fmt.Fprintf(conn, s)
}

func sendClickVal(conn net.Conn, pos int, action int, clickable string, val float64) {
	fmt.Fprintf(conn, "[1,%d,%d,\"%s\",%.3f]\n", pos, action, clickable, val)
}

func handleSwitch(conn net.Conn, switchState fpanels.SwitchState) {
	for _, routing := range conf.switchRouting {
		if switchState == routing.trigger {
			if routing.dir == BTN_SET {
				sendClickVal(conn, routing.mouse, routing.dir, routing.clickable, routing.value)
			} else {
				sendClick(conn, routing.mouse, routing.dir, routing.clickable)
			}
		}
	}
}

func main() {
	var err error
	var radioSwitches chan fpanels.SwitchState
	var multiSwitches chan fpanels.SwitchState
	var switchSwitches chan fpanels.SwitchState

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
	conn, err := net.Dial("tcp", server)
	if err != nil {
		log.Fatal("%v", err)
		return
	}
	log.Print("Connect")
	go readJSON(conn)

	SubscribeDisplay(conn, "RSBN_NAV_Chan", fpanels.RADIO, fpanels.ACTIVE_1, "%2.f***")
	SubscribeDisplay(conn, "RSBN_LAND_Chan", fpanels.RADIO, fpanels.ACTIVE_1, "***%2.f")
	if radioPanel != nil {
		radioSwitches = radioPanel.WatchSwitches()
	}
	if switchPanel != nil {
		switchSwitches = switchPanel.WatchSwitches()
	}
	if multiPanel != nil {
		switchSwitches = multiPanel.WatchSwitches()
	}
	// FIX: preiodically try to connect to unconnected panels
	var switchState fpanels.SwitchState
	for {
		select {
		case switchState = <-radioSwitches:
		case switchState = <-switchSwitches:
		case switchState = <-multiSwitches:
		}
		handleSwitch(conn, switchState)
		log.Printf("%v", switchState)
	}
}
