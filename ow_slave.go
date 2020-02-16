package main

import (
	"fmt"
	"strings"
	"strconv"
)



type OwSlave struct {
	Name		string
	Id			uint64
	HexId		string		`json:",omitempty"`
	Value		float64

	Thermostat		*Thermo
}

func (slave *OwSlave) SetFromInt(input uint64) {
	slave.Value = float64(input) / 1000
}

func (slave *OwSlave) InitId() bool {
	if len(slave.HexId) > 1 {
		idSlice := strings.Split(slave.HexId, "-")
		id, err := strconv.ParseUint(idSlice[len(idSlice) - 1], 16, 64)
		if err == nil {
			slave.Id = id
			return true
		}
	}

	return false
}

func (slave *OwSlave) InitThermo() error {
	if slave.Thermostat == nil {
		return nil 
	}

	if slave.Thermostat.Gpio == 0 {
		slave.Thermostat = nil
		return fmt.Errorf("OwSlave InitThermo: thermostat found, but no gpio config - removing")
	}

	slave.Thermostat.Sensor = slave
	if slave.Thermostat.Hysteresis == 0 {
		slave.Thermostat.Hysteresis = 0.5
	}
	if slave.Thermostat.Max == 0 {
		slave.Thermostat.Max = 40
	}

	err := slave.Thermostat.ReadState()
	if err != nil {
		return fmt.Errorf("ERROR OwSlave InitThermo ReadState failed:\n%w", err)
	}

	return nil
}