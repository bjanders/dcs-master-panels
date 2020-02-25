package main

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/bjanders/fpanels"
)

type config struct {
	server         string
	aircraft       string
	displayRouting []*DisplayRouting
	devCmdRouting  []*SwitchRouting
}

func parseInput(line string) (*DisplayRouting, error) {
	var token string
	var err error
	var routing = DisplayRouting{}

	scanner := bufio.NewScanner(strings.NewReader(line))
	scanner.Split(bufio.ScanWords)

	if scanner.Scan() == false {
		return nil, errors.New("Expected panel name or '['")
	}

	if scanner.Text() == "[" {
		if scanner.Scan() == false {
			return nil, errors.New("Expected panel name")
		}
		routing.cond, err = parseSwitchState(scanner)
		if err != nil {
			return nil, err
		}
		if scanner.Scan() == false {
			return nil, errors.New("Expected ']'")
		}
		if scanner.Text() != "]" {
			return nil, errors.New("Expected ']'")
		}
		if scanner.Scan() == false {
			return nil, errors.New("Expected panel name")
		}
	}
	// Panel
	token = scanner.Text()
	routing.panel, err = fpanels.PanelIDString(token)
	if err != nil {
		return nil, errors.New("Unknown panel name: " + token)
	}

	// Display or LEDs
	if scanner.Scan() == false {
		return nil, errors.New("Expected display or LED name")
	}
	token = scanner.Text()
	routing.display, err = fpanels.DisplayIDString(token)
	if err != nil {
		routing.leds, err = fpanels.LEDString(token)
		if err != nil {
			return nil, errors.New("Unknown display or LED name")
		}
		if scanner.Scan() == false {
			return nil, errors.New("Expexted '<-'")
		}
		token = scanner.Text()
		if token != "<-" {
			return nil, errors.New("Expected '<-'")
		}
	} else {
		if scanner.Scan() == false {
			return nil, errors.New("Expexted '<-' or format string")
		}
		token = scanner.Text()
		if token != "<-" {
			if token[0] != '"' || token[len(token)-1] != '"' {
				return nil, errors.New("Expected format string")
			}
			routing.format = token[1 : len(token)-1]
			if scanner.Scan() == false {
				return nil, errors.New("Expexted '<-'")
			}
			token = scanner.Text()
			if token != "<-" {
				return nil, errors.New("Expected '<-'")
			}
		} else {
			routing.format = "%.f"
		}
	}
	if scanner.Scan() == false {
		return nil, errors.New("Expected gauge name")
	}
	routing.gaugeName = scanner.Text()
	if scanner.Scan() == false {
		return &routing, nil
	}
	// get precission
	routing.prec, err = strconv.Atoi(scanner.Text())
	if err != nil {
		return nil, errors.New("Unable to parse precission")
	}
	return &routing, nil

}

func parseSwitchState(scanner *bufio.Scanner) (*fpanels.SwitchState, error) {
	var switchState fpanels.SwitchState
	var err error

	token := scanner.Text()
	switchState.Panel, err = fpanels.PanelIDString(token)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Unknown panel name '%s'", token))
	}

	// Switch
	if scanner.Scan() == false {
		return nil, errors.New("Expected switch name")
	}
	token = scanner.Text()
	split := strings.Split(token, "=")
	switchState.Switch, err = fpanels.SwitchIDString(split[0])
	if err != nil {
		return nil, errors.New("Unknown switch name")
	}
	if len(split) == 2 {
		val, err := strconv.ParseInt(split[1], 10, 0)
		if err != nil {
			return nil, errors.New("Unable to parse switchState value")
		}
		switchState.On = val == 1
	} else {
		switchState.On = true
	}
	return &switchState, nil
}

func parseCmdOutput(line string) ([]*SwitchRouting, error) {
	var routing = &SwitchRouting{}
	var routingList []*SwitchRouting
	var err error
	var token string

	scanner := bufio.NewScanner(strings.NewReader(line))
	scanner.Split(bufio.ScanWords)

	// Panel
	if scanner.Scan() == false {
		return routingList, errors.New("Expected panel name or '['")
	}

	if scanner.Text() == "[" {
		if scanner.Scan() == false {
			return routingList, errors.New("Expected panel name")
		}
		routing.cond, err = parseSwitchState(scanner)
		if err != nil {
			return routingList, err
		}
		if scanner.Scan() == false {
			return routingList, errors.New("Expected ']', found nothing")
		}
		token = scanner.Text()
		if token != "]" {
			return routingList, errors.New(fmt.Sprintf("Expected ']', not '%s'", token))
		}
		if scanner.Scan() == false {
			return routingList, errors.New("Expected panel name")
		}
	}

	routing.trigger, err = parseSwitchState(scanner)
	if err != nil {
		return routingList, err
	}

	// "->"
	if scanner.Scan() == false || scanner.Text() != "->" {
		return routingList, errors.New("Expected '->'")
	}
	// device name
	if scanner.Scan() == false {
		return routingList, errors.New("Expected device name")
	}
	routing.cmd.Dev = scanner.Text()

	// command name
	if scanner.Scan() == false {
		return routingList, errors.New("Expected command name")
	}
	routing.cmd.Cmd = scanner.Text()

	if scanner.Scan() == false {
		routing.cmd.Val = 1.0
		routingList = append(routingList, routing)
		routing = routing.copy()
		routing.trigger.On = false
		routing.cmd.Val = 0.0
		routingList = append(routingList, routing)
		return routingList, nil
	}

	// get value to set
	routing.cmd.Val, err = strconv.ParseFloat(scanner.Text(), 64)
	if err != nil {
		return routingList, errors.New("Unable to parse value to set")
	}

	routingList = append(routingList, routing)

	if scanner.Scan() == false {
		return routingList, nil
	}
	routing = routing.copy()
	routing.trigger.On = false
	routing.cmd.Val, err = strconv.ParseFloat(scanner.Text(), 64)
	if err != nil {
		return routingList, errors.New("Unable to parse value to set")
	}
	routingList = append(routingList, routing)
	return routingList, nil
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
	//log.Printf("Server set to %s", conf.server)
}

func (config *config) setAircraft(aircraft string) {
	var line string
	var plane string

	conf.displayRouting = nil
	conf.devCmdRouting = nil
	config.aircraft = aircraft
	plane = strings.Replace(conf.aircraft, " ", "_", -1)
	plane = strings.ToLower(plane) + ".conf"
	file, err := os.Open(plane)
	if err != nil {
		log.Print(err)
		return
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	lineNumber := 1
	for scanner.Scan() {
		line = scanner.Text()
		switch {
		case strings.Contains(line, "->"):
			devRouting, err := parseCmdOutput(line)
			if err != nil {
				log.Printf("Line %d: %s", lineNumber, err)
			} else {
				conf.devCmdRouting = append(conf.devCmdRouting, devRouting...)
			}
		case strings.Contains(line, "<-"):
			displayRouting, err := parseInput(line)
			if err != nil {
				log.Printf("Line %d, %s", lineNumber, err)
			} else {
				displayRouting.gaugeID = gaugeCount
				conf.displayRouting = append(conf.displayRouting, displayRouting)
				gaugeCount++
			}
		}
		lineNumber++
	}
}
