package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
)

const (
	CMD_AIRCRAFT  = 0
	CMD_DEVCMD    = 1
	CMD_SUBSCRIBE = 2
)

type DCS struct {
	server string
	conn   net.Conn
	Ch     DCSChannels
}

type DCSChannels struct {
	gauges    chan Gauge
	aircraft  chan string
	connected chan bool
}

type DevCmd struct {
	Dev string
	Cmd string
	Val float64
}

type Gauge struct {
	Id  int
	Val float64
}

func (dcs *DCS) Connect(server string) error {
	var err error
	dcs.server = server
	dcs.conn, err = net.Dial("tcp", server)
	if err != nil {
		return err
	}
	dcs.Ch.aircraft = make(chan string)
	dcs.Ch.gauges = make(chan Gauge)
	dcs.Ch.connected = make(chan bool)
	go dcs.readJSON()
	return nil
}

func (dcs *DCS) readJSON() {
	decoder := json.NewDecoder(dcs.conn)
	var m []interface{}
	for {
		err := decoder.Decode(&m)
		if err != nil {
			if err.Error() != "EOF" {
				log.Fatal(err)
			}
			dcs.Ch.connected <- false
			return
		}
		cmd := int(m[0].(float64))
		//log.Printf("command %v", cmd)
		switch cmd {
		case CMD_AIRCRAFT:
			aircraft := m[1].(string)
			//log.Printf("Got aircraft %s", aircraft)
			dcs.Ch.aircraft <- aircraft
		case CMD_SUBSCRIBE:
			gauges := decodeGauges(m[1:])
			for _, g := range gauges {
				dcs.Ch.gauges <- g
			}
		}
	}
}

func decodeGauges(data []interface{}) []Gauge {
	var gauges []Gauge
	for _, g := range data {
		list := g.([]interface{})
		gauge := Gauge{int(list[0].(float64)), list[1].(float64)}
		gauges = append(gauges, gauge)
		//log.Printf("gauge %v", gauge)
	}
	return gauges
}

func quote(str string) string {
	return fmt.Sprintf("\"%s\"", str)
}

func (dcs *DCS) sendSubscribe(gauge string, id int, prec int) {
	//log.Print("Subscribing to ", gauge)
	fmt.Fprintf(dcs.conn, "[%d,%s,%d,%d]\n", CMD_SUBSCRIBE, quote(gauge), id, prec)
}

func (dcs *DCS) sendDevCmd(devCmd *DevCmd) {
	s := fmt.Sprintf("[%d,%s,%s,%.3f]\n", CMD_DEVCMD, quote(devCmd.Dev), quote(devCmd.Cmd), devCmd.Val)
	//log.Printf(s)
	fmt.Fprintf(dcs.conn, s)
}
