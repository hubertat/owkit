package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type OwSet struct {
	Path        string `json:",omitempty"`
	SlavePrefix string `json:",omitempty"`
	Debug       bool   `json:",omitempty"`

	Sensors []*OwSlave `json:",omitempty"`

	LogInflux *InfluxWriter `json:",omitempty"`
	SendHttp  *HttpWriter   `json:",omitempty"`

	Server      *Server           `json:",omitempty"`
	OffPeak     *OffPeak          `json:",omitempty"`
	EnergyPanel *VictronGridMeter `json:",omitempty"`

	RefreshSeconds int `json:",omitempty"`
	Updated        time.Time

	refreshInterval time.Duration
	tick            *time.Ticker
	blocker         sync.Mutex
}

func (os *OwSet) LogDebug(message string) {
	if !os.Debug {
		return
	}

	log.Printf("? Debug:\n%s\n", message)
}

func (os *OwSet) Log(message string) {
	log.Printf("%s\n", message)
}

func (os *OwSet) CheckIfSet() bool {
	if len(os.Path) == 0 && len(os.SlavePrefix) == 0 {
		return false
	}

	return true
}

func (os *OwSet) Set(configPath ...string) error {

	os.Path = "/sys/bus/w1/devices"
	os.SlavePrefix = "28-"

	if os.RefreshSeconds == 0 {
		os.RefreshSeconds = 15
	}
	os.refreshInterval, _ = time.ParseDuration(fmt.Sprintf("%ds", os.RefreshSeconds))

	if len(configPath) == 0 {
		return nil
	}

	configFile, err := ioutil.ReadFile(configPath[0])
	if err != nil {
		return fmt.Errorf("OwSet Set: error openning config file:\n %w", err)
	}

	os.Log(fmt.Sprintf("Loading config form file: %s\n", configPath))
	err = json.Unmarshal([]byte(configFile), os)
	if err != nil {
		return fmt.Errorf("OwSet Set: error reading config(json) into OwSet:\n %w", err)
	}

	if os.Server != nil {
		os.Server.set = os
	}

	for _, slave := range os.Sensors {
		slave.InitId()

		err = slave.InitThermo()
		if err != nil {
			return fmt.Errorf("OwSet Set | error initializing thermostat:\n%v", err)
		}
	}

	return nil
}

func (os *OwSet) InitSlaves(settings ...string) error {
	os.blocker.Lock()
	defer os.blocker.Unlock()

	err := os.Set(settings...)
	if err != nil {
		return fmt.Errorf("OwSet InitSlaves: Set failed:\n%v", err)
	}

	if !os.CheckIfSet() {
		fmt.Printf("%+v\n", os)
		return fmt.Errorf("OwSet InitSlaves: set was not set properly (should ran OwSet.Set)")
	}

	devs, err := ioutil.ReadDir(os.Path)
	if err != nil {
		return fmt.Errorf("OwSet InitSlaves: error reading dir (%s):\n%w", os.Path, err)
	}

	var zeroId, alreadyHere *OwSlave

	for _, dev := range devs {
		if strings.HasPrefix(dev.Name(), os.SlavePrefix) {
			wslave, err := ioutil.ReadFile(filepath.Join(os.Path, dev.Name(), "w1_slave"))
			if err == nil {
				id, errId := strconv.ParseUint(strings.TrimPrefix(dev.Name(), os.SlavePrefix), 16, 64)
				val, errVal := strconv.ParseUint(string(wslave[69:74]), 10, 64)
				if string(wslave[36:39]) == "YES" && errId == nil && errVal == nil {

					alreadyHere = os.GetSlaveById(id)
					if alreadyHere != nil {

						alreadyHere.SetFromInt(val)
					} else {

						zeroId = os.GetSlaveById(0)
						if zeroId == nil {
							zeroId = &OwSlave{Id: id}
						}
						zeroId.SetFromInt(val)
						os.Sensors = append(os.Sensors, zeroId)
					}
				}
			}
		}

	}
	os.Updated = time.Now()

	return nil
}

func (os *OwSet) GetSlaveById(id uint64) *OwSlave {
	for _, slave := range os.Sensors {
		if slave.Id == id {
			return slave
		}
	}

	return nil
}

func (os *OwSet) GetSlaveByName(name string) *OwSlave {
	for _, slave := range os.Sensors {
		if slave.Name == name {
			return slave
		}
	}

	return nil
}

func (os *OwSet) GetSlave(ident string) *OwSlave {
	var slave *OwSlave
	intId, err := strconv.ParseUint(ident, 10, 64)
	if err != nil {
		slave = os.GetSlaveById(intId)
		if slave != nil {
			return slave
		}
	}

	return os.GetSlaveByName(ident)
}

func (os *OwSet) RefreshAll() error {

	if os.Updated.IsZero() {
		err := os.InitSlaves()
		if err != nil {
			return fmt.Errorf("OwSet RefreshAll: error during forced InitSlaves:\n%w", err)
		}
	}

	os.blocker.Lock()
	defer os.blocker.Unlock()

	for _, slave := range os.Sensors {

		wslave, err := ioutil.ReadFile(filepath.Join(os.Path, fmt.Sprintf("%s%012x", os.SlavePrefix, slave.Id), "w1_slave"))
		if err != nil {
			return fmt.Errorf("OwSet RefreshAll: error reading file (id %x):\n%w\naborting", slave.Id, err)
		}
		if string(wslave[36:39]) != "YES" {
			return fmt.Errorf("OwSet RefreshAll: (id %x) crc not YES, aborting", slave.Id)
		}
		wslaveRaw := string(wslave)
		tempSlice := strings.Split(wslaveRaw, "t=")
		if len(tempSlice) != 2 {
			return fmt.Errorf("OwSet RefreshAll: (id %x) t= not found or found multiple times, aborting", slave.Id)
		}
		if len(tempSlice[1]) == 0 {
			return fmt.Errorf("OwSet RefreshAll: (id %x) empty value, aborting", slave.Id)
		}
		tempStr := strings.TrimSpace(tempSlice[1])
		val, err := strconv.ParseUint(tempStr, 10, 64)
		if err != nil {
			return fmt.Errorf("OwSet RefreshAll: error parsing value (%s) (id %x):\n%w\naborting", tempStr, slave.Id, err)
		}

		slave.SetFromInt(val)
	}
	os.Updated = time.Now()

	return nil
}

func (os *OwSet) cycling() {
	var err error
	for {
		select {
		case <-os.tick.C:
			err = os.RefreshAll()
			if err != nil {
				log.Printf("ERROR [in OwSet] during refreshing during cycling:\n%v", err)
			} else {
				var offPeakHeatUp, energyPanelHeatUp bool
				if os.OffPeak != nil {
					os.LogDebug("OffPeak enabled [OwSet], checking state")
					offPeakHeatUp = os.OffPeak.Check()
				}
				if os.EnergyPanel != nil {
					os.LogDebug("EnergyPanel enabled, ticking and checking")
					err = os.EnergyPanel.Tick()
					if err != nil {
						os.Log(fmt.Sprintf("Received error from EnergyPanel.Tick(): %v", err))
					} else {
						os.LogDebug(os.EnergyPanel.GetDebugString())
						energyPanelHeatUp = os.EnergyPanel.CheckAvPowerLimit()
					}

				}

				if offPeakHeatUp {
					os.LogDebug("Received OffPeak, setting heat up mode")
				}
				if energyPanelHeatUp {
					os.LogDebug("Received OK Power Limit from Energy Panel, setting heat up mode")
				}
				for _, slave := range os.Sensors {
					if slave.Thermostat != nil {
						os.LogDebug(fmt.Sprintf("Thermostat found, setting heatUpMode: %v", (energyPanelHeatUp || offPeakHeatUp)))
						slave.Thermostat.HeatUpMode = energyPanelHeatUp || offPeakHeatUp
					}
				}
				os.PrintAll()
				os.RunThermostats()

				if os.LogInflux != nil {
					log.Print("Sendings readouts to influx")
					err = os.LogInflux.Send(os.Sensors)
					if err != nil {
						log.Printf("ERROR | OwSet | sending LogInflux:\n%v", err)
					}
				}
				if os.SendHttp != nil {
					log.Print("Sending values through Http")
					err = os.SendHttp.Send(os.Sensors)
					if err != nil {
						log.Printf("ERROR | OwSet | sending values by Http:\n%v", err)
					}
				}
			}
		}
	}
}
func (os *OwSet) StartCycling() {
	os.tick = time.NewTicker(os.refreshInterval)
	go os.cycling()
}

func (os *OwSet) PrintAll() {
	freshness := time.Since(os.Updated)
	log.Printf("Printing all sensors, last refresh %fs ago\n", freshness.Seconds())
	fmt.Printf("id\t\tname\t\tvalue\t\tthermo?\t\tsetpoint\tstate\n")
	for _, slave := range os.Sensors {
		fmt.Printf("%s\t\t%x\t\t%.2f\t\t", slave.Name, slave.Id, slave.Value)
		if slave.Thermostat == nil {
			fmt.Printf("no\t\t-\t-\n")
		} else {
			fmt.Printf("yes\t\t%f\t%t\n", slave.Thermostat.Setpoint, slave.Thermostat.IsOn)
		}
	}
}

func (os *OwSet) RunThermostats() {
	var err error

	for _, slave := range os.Sensors {
		if slave.Thermostat != nil {
			err = slave.Thermostat.Run()
			if err != nil {
				log.Printf("ERROR OwSet RunThermostats failed (for %v):\n%v", slave.Name, err)
			}
		}
	}
}
