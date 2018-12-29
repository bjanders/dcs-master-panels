package main

import (
	"encoding/json"
	"fmt"
	"github.com/bjanders/fpanels"
	"log"
	"net"
)

const server = "127.0.0.1:8888"

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

type Message struct {
	Cmd    int
	Gauges []Gauge
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
	gaugeId int
	gaugeName string
	cond fpanels.SwitchState
	panel fpanels.PanelId
	display fpanels.DisplayId
	format string
}

var displayRouting []DisplayRouting

//func addDisplay()

type SwitchRouting struct {
	trigger fpanels.SwitchState
	cond fpanels.SwitchState
	mouse int
	dir int
	clickable string
}

var switchRouting []SwitchRouting
var aircraft string

// siwthcRouting int ->  mousee, dir, clickable
// radioswithrouting switchRouting
// multiswithRouting


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
		for _, routing := range displayRouting {
			if gauge.Id == routing.gaugeId {
			switch routing.panel {
			case fpanels.RADIO:
				radioPanel.DisplayString(routing.display, fmt.Sprintf(routing.format, gauge.Val))
			}
			}
		}
	}
}

func handleSubscribe(data []interface{}) {
	gauges := decodeGauges(data)
	log.Printf("%v", gauges)
}

func handleAircraft(data []interface{}) {
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

func sendSubscribe(conn net.Conn, gauge string) {
	fmt.Fprintf(conn, "[%d,\"%s\"]\n", CMD_SUBSCRIBE, gauge)
}

func sendClick(conn net.Conn, pos int, action int, clickable string) {
	fmt.Fprintf(conn, "[1,%d,%d,\"%s\"]\n", pos, action, clickable)
}

func handleRadio(conn net.Conn, switchState fpanels.SwitchState) {
	switch switchState.Switch {
	case fpanels.ENC1_CW_1:
		sendClick(conn, MOUSE_RIGHT, BTN_DOWN, "CHANNEL_NAVIGATION")
	case fpanels.ENC1_CCW_1:
		sendClick(conn, MOUSE_LEFT, BTN_DOWN, "CHANNEL_NAVIGATION")
	}
}

func main() {
	//radio, _ := fpanels.NewRadioPanel()
	//if err != nil {
	//	log.Fatal("%v", err)
	//	return
	//}
	conn, err := net.Dial("tcp", server)
	if err != nil {
		log.Fatal("%v", err)
		return
	}
	log.Print("Connect")
	go readJSON(conn)

	sendSubscribe(conn, "RSBN_NAV_Chan")
	//sendSubscribe(conn, "IAS")
	//sendSubscribe(conn, "Acceleration")
	//radioSwitches := radio.WatchSwitches()
	//var switchState fpanels.SwitchState
	//for {
	//	select {
	//	case switchState = <-radioSwitches:
	//		handleRadio(conn, switchState)
	//	}
	//}
	for {
	}
}
