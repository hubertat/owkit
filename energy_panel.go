package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/goburrow/modbus"
)

type VictronGridMeter struct {
	ConnectionString string
	HoldMinutes      int
	PowerLevel       int

	readouts []struct {
		When       time.Time
		TotalPower int
	}

	raw struct {
		Power [3]int16
	} `json:"-"`
}

func (vgm *VictronGridMeter) ReadBytes(result []byte) (err error) {
	reader := bytes.NewReader(result)
	err = binary.Read(reader, binary.BigEndian, &vgm.raw)

	return
}

func (vgm *VictronGridMeter) readModbus() (err error) {
	slaveId := 31
	dataAddr := 2600
	dataCount := 3

	handler := modbus.NewTCPClientHandler(vgm.ConnectionString)
	handler.Timeout = 2 * time.Second
	handler.SlaveId = byte(slaveId)

	err = handler.Connect()
	if err != nil {
		return
	}
	defer handler.Close()

	client := modbus.NewClient(handler)
	result, err := client.ReadHoldingRegisters(uint16(dataAddr), uint16(dataCount))
	if err != nil {
		return
	}

	err = vgm.ReadBytes(result)

	return
}

func (vgm *VictronGridMeter) cleanOldReadouts() {
	holdDuration := time.Minute * time.Duration(vgm.HoldMinutes)
	for ix, readout := range vgm.readouts {
		if time.Since(readout.When) > holdDuration {
			if (ix + 1) == len(vgm.readouts) {
				vgm.readouts = vgm.readouts[:ix]
			} else {
				vgm.readouts = append(vgm.readouts[:ix], vgm.readouts[ix+1:]...)
			}
		}
	}
}

func (vgm *VictronGridMeter) GetAveragePower() int {
	if len(vgm.readouts) == 0 {
		return 0
	}

	var powerSum int
	for _, readout := range vgm.readouts {
		powerSum += readout.TotalPower
	}

	return powerSum / len(vgm.readouts)
}

func (vgm *VictronGridMeter) Tick() error {
	err := vgm.readModbus()
	if err != nil {
		return err
	}

	vgm.cleanOldReadouts()

	var totalP int
	for _, p := range vgm.raw.Power {
		totalP += int(p)
	}

	vgm.readouts = append(vgm.readouts, struct {
		When       time.Time
		TotalPower int
	}{When: time.Now(), TotalPower: totalP})

	return nil
}

func (vgm *VictronGridMeter) CheckAvPowerLimit() bool {
	return vgm.GetAveragePower() < (-1 * vgm.PowerLevel)
}

func (vgm *VictronGridMeter) GetDebugString() string {
	return fmt.Sprintf("VictronGridMeter:: raw energy data: %v; readouts count: %d; average total power: %d", vgm.raw, len(vgm.readouts), vgm.GetAveragePower())
}
