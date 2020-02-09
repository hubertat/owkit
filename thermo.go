package main

import (
	"fmt"
	"log"
	"github.com/stianeikeland/go-rpio"
)

type Thermo struct {
	Gpio	int
	Invert	bool

	isOn bool

	Setpoint, Hysteresis, Min, Max float64
	Sensor		*OwSlave		`json:"-"`
}

func (th *Thermo) Run() (err error) {
	if th.isOn {
		if th.Sensor.Value > (th.Setpoint + th.Hysteresis) {
			err = th.Set(false)
			return
		}
	} else {
		if th.Sensor.Value < (th.Setpoint - th.Hysteresis) {
			err = th.Set(true)
			return
		}
	}
	return
}

func (th *Thermo) ReadState() error {
	log.Print("Thermo Set: checking gpio state")
	err := rpio.Open()
	if err != nil {
		return fmt.Errorf("Thermo Set: opening rpio failed:\n%w", err)
	}

	defer rpio.Close()

	pin := rpio.Pin(th.Gpio)
	pin.Output()
	
	state := true
	if pin.Read() == rpio.High {
		state = false	
	}
	if th.Invert {
		th.isOn = !state
	} else {
		th.isOn = state
	}

	return nil
}

func (th *Thermo) Set(state bool) (error) {
	log.Print("Thermo Set: received [%v] request, running.", state)
	err := rpio.Open()
	if err != nil {
		return fmt.Errorf("Thermo Set: opening rpio failed:\n%w", err)
	}

	defer rpio.Close()

	pin := rpio.Pin(th.Gpio)
	pin.Output()

	th.isOn = state

	if th.Invert {
		state = !state
	}

	if state {
		pin.Low()
	} else {
		pin.High()
	}

	return nil
}