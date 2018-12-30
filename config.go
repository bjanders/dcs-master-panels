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

func parseOutput(line string) ([]SwitchRouting, error) {
	var trigger = fpanels.SwitchState{}
	var routing = SwitchRouting{}
	var routingList []SwitchRouting
	var err error

	scanner := bufio.NewScanner(strings.NewReader(line))
	scanner.Split(bufio.ScanWords)

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
		routing2 := routing
		routing2.trigger.Value = 0
		routing2.value = 0
		routingList = append(routingList, routing2)
		return routingList, nil
	}
	// FIX: scan value
	return append(routingList, routing), nil
}

func (config *config) read() {
	var line string
	var plane string

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
		if strings.Contains(line, "->") {
			switchRouting, err := parseOutput(line)
			if err != nil {
				log.Print(err)
			} else {
				for _, r := range switchRouting {
					conf.switchRouting = append(conf.switchRouting, r)
				}
			}
		}
	}
	for _, r := range conf.switchRouting {
		log.Print(r)
	}
}
