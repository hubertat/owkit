package main

import (
	"fmt"
	"log"
	"github.com/stianeikeland/go-rpio"
)

type Thermo struct {
	Gpio		int
	Invert		bool

	IsOn 			bool
	HeatUpMode		bool

	Setpoint, Hysteresis, Min, Max, HeatUp 			float64
	
	Sensor			*OwSlave		`json:"-"`
}

func (th *Thermo) Run() (err error) {

	if th.IsOn {
		if th.Sensor.Value > (th.GetSetpoint() + th.Hysteresis) {
			err = th.Set(false)
		}
	} else {
		if th.Sensor.Value < (th.GetSetpoint() - th.Hysteresis) {
			err = th.Set(true)
		}
	}
	return
}

func (th *Thermo) GetSetpoint() (setpoint float64) {
	setpoint = th.Setpoint

	if th.HeatUpMode {
		if th.HeatUp + th.Setpoint > th.Max {
			setpoint = th.Max
		} else {
			setpoint += th.HeatUp
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
		th.IsOn = !state
	} else {
		th.IsOn = state
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

	th.IsOn = state

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

func (th *Thermo) SetHeatUp(state ...bool) {
	if len(state) > 1 {
		th.HeatUpMode = state[0]
	} else {
		th.HeatUpMode = true
	}
}

func (th *Thermo) CheckIfOn() uint {
	if th.IsOn {
		return 1
	} else {
		return 0
	}
}

func (th *Thermo) CheckIfHeatUp() uint {
	if th.HeatUpMode {
		return 1
	} else {
		return 0
	}
}

// type OffPeak struct {
// 	StartWeekday, StopWeekday	time.Weekday
// 	StartHour, StopHour			int
// 	StartMinute, StopMinute		int
// }


// func (op *OffPeak) CheckIfInside(when time.Time) (result bool) {
// 	// check if Weekday matters at all
// 	if op.StartWeekday == op.StopWeekday {
// 		dayStart := when.Day()
// 		dayStop := when.Day()
// 	} else {

// 	}
// 	tStart := time.Date(when.Year(), when.Month(), dayStart, op.StartHour, op.StartMinute, 0, 0, when.Location())
// 	tStop := time.Date(when.Year(), when.Month(), dayStop, op.StopHour, op.StartMinute, 0, 0, when.Location())
	
// 	// check if stop is after start
// 	if tStop.After(tStart) {
// 		if when.After(tStart) && when.Before(tStop) {
// 			result = true
// 		}
// 	} else {
// 		// it means that offPeak period is not contained in one day, we should require only one condition
// 		if when.After(tStart) || when.Before(tStop) {
// 			result = true
// 		}
// 	}

// 	return
// }
