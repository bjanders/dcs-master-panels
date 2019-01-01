package main

import (
	"bufio"
	"errors"
	"github.com/bjanders/fpanels"
	"log"
	"os"
	"strconv"
	"strings"
)

type config struct {
	server         string
	aircraft       string
	switchRouting  []SwitchRouting
	displayRouting []DisplayRouting
}

func parseInput(line string) (DisplayRouting, error) {
	var token string
	var err error
	var routing = DisplayRouting{}

	scanner := bufio.NewScanner(strings.NewReader(line))
	scanner.Split(bufio.ScanWords)
	// FIX: read optional condition "[panel switch]"

	// Panel
	if scanner.Scan() == false {
		return routing, errors.New("Expected panel name")
	}
	token = scanner.Text()
	routing.panel, err = fpanels.PanelIdString(token)
	if err != nil {
		return routing, errors.New("Unknown panel name: " + token)
	}

	// Display or LEDs
	if scanner.Scan() == false {
		return routing, errors.New("Expected display or LED name")
	}
	token = scanner.Text()
	routing.display, err = fpanels.DisplayIdString(token)
	if err != nil {
		routing.leds, err = fpanels.LEDString(token)
		if err != nil {
			return routing, errors.New("Unknown display or LED name")
		}
		log.Printf("LEDs %0x", routing.leds)
		if scanner.Scan() == false {
			return routing, errors.New("Expexted '<-'")
		}
		token = scanner.Text()
		if token != "<-" {
			return routing, errors.New("Expected '<-'")
		}
	} else {
		if scanner.Scan() == false {
			return routing, errors.New("Expexted '<-' or format string")
		}
		token = scanner.Text()
		if token != "<-" {
			if token[0] != '"' || token[len(token)-1] != '"' {
				return routing, errors.New("Expected format string")
			}
			routing.format = token[1 : len(token)-1]
			log.Print("routing format ", routing.format)
			if scanner.Scan() == false {
				return routing, errors.New("Expexted '<-'")
			}
			token = scanner.Text()
			if token != "<-" {
				return routing, errors.New("Expected '<-'")
			}
		}
	}
	if scanner.Scan() == false {
		return routing, errors.New("Expected gauge name")
	}
	routing.gaugeName = scanner.Text()
	return routing, nil

}

func parseOutput(line string) ([]SwitchRouting, error) {
	var trigger = fpanels.SwitchState{}
	var routing = SwitchRouting{}
	var routingList []SwitchRouting
	var err error

	scanner := bufio.NewScanner(strings.NewReader(line))
	scanner.Split(bufio.ScanWords)
	// FIX: read optional condition "[panel switch]"

	// Panel
	if scanner.Scan() == false {
		return routingList, errors.New("Expected panel name")
	}
	token := scanner.Text()
	trigger.Panel, err = fpanels.PanelIdString(token)
	if err != nil {
		return routingList, errors.New("Unknown panel name")
	}

	// Switch
	if scanner.Scan() == false {
		return routingList, errors.New("Expected switch name")
	}
	token = scanner.Text()
	split := strings.Split(token, "=")
	trigger.Switch, err = fpanels.SwitchIdString(split[0])
	if err != nil {
		return routingList, errors.New("Unknown switch name")
	}
	if len(split) == 2 {
		val, err := strconv.ParseInt(split[1], 10, 0)
		if err != nil {
			return routingList, errors.New("Unable to parse trigger value")
		}
		trigger.Value = uint(val)
	} else {
		trigger.Value = 1
	}

	routing.trigger = trigger

	// "->"
	if scanner.Scan() == false || scanner.Text() != "->" {
		return routingList, errors.New("Expected '->'")
	}

	// clickable name
	if scanner.Scan() == false {
		return routingList, errors.New("Expected clickable name")
	}
	routing.clickable = scanner.Text()

	// left or right button
	// FIX: also accept 'set' and assume 'left'
	if scanner.Scan() == false {
		return routingList, errors.New("Expected 'left' or 'right'")
	}
	switch strings.ToLower(scanner.Text()) {
	case "left":
		routing.mouse = MOUSE_LEFT
	case "right":
		routing.mouse = MOUSE_RIGHT
	default:
		return routingList, errors.New("Expected 'left' or 'right'")
	}
	if scanner.Scan() == false {
		routing.dir = BTN_DOWN
		return append(routingList, routing), nil
	}

	// up, down or set
	switch strings.ToLower(scanner.Text()) {
	case "down":
		routing.dir = BTN_DOWN
		return append(routingList, routing), nil
	case "up":
		routing.dir = BTN_UP
		return append(routingList, routing), nil
	case "set":
		routing.dir = BTN_SET
	default:
		return append(routingList, routing), errors.New("Expected 'down', 'up' or 'set'")
	}

	// set value. If no value given automatically create two
	if scanner.Scan() == false {
		routing.value = 1
		routingList = append(routingList, routing)
		routing.trigger.Value = 0
		routing.value = 0
		routingList = append(routingList, routing)
		return routingList, nil
	}

	// get value to set
	routing.value, err = strconv.ParseFloat(scanner.Text(), 64)
	if err != nil {
		return routingList, errors.New("Unable to parse value to set")
	}

	return append(routingList, routing), nil
}

func (config *config) getServer() {
	file, err := os.Open("server.conf")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Scan()
	config.server = scanner.Text()
	log.Printf("Server set to %s", conf.server)
}

func (config *config) setAircraft(aircraft string) {
	var line string
	var plane string

	config.aircraft = aircraft
	plane = strings.Replace(conf.aircraft, " ", "_", -1)
	plane = strings.ToLower(plane) + ".conf"
	file, err := os.Open(plane)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line = scanner.Text()
		switch {
		case strings.Contains(line, "->"):
			switchRouting, err := parseOutput(line)
			if err != nil {
				log.Print(err)
			} else {
				conf.switchRouting = append(conf.switchRouting, switchRouting...)
			}
		case strings.Contains(line, "<-"):
			displayRouting, err := parseInput(line)
			if err != nil {
				log.Print(err)
			} else {
				displayRouting.gaugeId = gaugeCount
				conf.displayRouting = append(conf.displayRouting, displayRouting)
				gaugeCount++
			}
		}
	}
	for _, r := range conf.switchRouting {
		log.Print(r)
	}
}
