package main

import (
	"fmt"
	"log"
	"io/ioutil"
	"strings"
	"strconv"
	"net/http"
	"path/filepath"
	"encoding/json"
	"time"
	"sync"
)

type OwSet struct {
	Path		string		`json:",omitempty"`
	SlavePrefix	string		`json:",omitempty"`

	Sensors		[]*OwSlave		`json:",omitempty"`
	LogInflux	*InfluxWriter	`json:",omitempty"`

	RefreshSeconds		int	`json:",omitempty"`
	
	updated				time.Time
	refreshInterval		time.Duration
	tick	       		*time.Ticker
	blocker				sync.Mutex
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

	err = json.Unmarshal([]byte(configFile), os)
	if err != nil {
		return fmt.Errorf("OwSet Set: error reading config(json) into OwSet:\n %w", err)
	}

	for _, slave := range os.Sensors {
		slave.InitThermo()
	}

	return nil
}

func (os *OwSet) InitSlaves(settings ...string) error {
	os.blocker.Lock()
	defer os.blocker.Unlock()

	err := os.Set(settings...)
	if err != nil {
		return fmt.Errorf("OwSet InitSlaves: Set failed:\n%w")
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
	os.updated = time.Now()
	
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

func (os *OwSet) RefreshAll() error {

	if os.updated.IsZero() {
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
		val, err := strconv.ParseUint(string(wslave[69:74]), 10, 64)
		if err != nil {
			return fmt.Errorf("OwSet RefreshAll: error parsing value (id %x):\n%w\naborting", slave.Id, err)
		}

		slave.SetFromInt(val)
	}
	os.updated = time.Now()

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
				os.PrintAll()
				os.RunThermostats()

				if os.LogInflux != nil {
					log.Print("Sendings readouts to influx")
					err = os.LogInflux.Send(os.Sensors)
					if err != nil {
						log.Printf("ERROR [in OwSet] sending LogInflux:\n%v", err)
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
	freshness := time.Since(os.updated)
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

func (os *OwSet) HandleSet(w http.ResponseWriter, r *http.Request) {
	setS := &OwSlave{}
    err := json.NewDecoder(r.Body).Decode(setS)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    for _, slave := range os.Sensors {
    	if slave.Id == setS.Id {
    		err := json.NewDecoder(r.Body).Decode(slave)
    		if err != nil {
    			http.Error(w, err.Error(), http.StatusBadRequest)	
    		}
    		return
    	}
    }

    http.Error(w, "Sensor not found", 404)
    return
}

func (os *OwSet) HandleState(w http.ResponseWriter, r *http.Request) {
	
	js, err := json.Marshal(os)
	
	if err != nil {
	http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func (os OwSet) HandleAllHeatUp(w http.ResponseWriter, r *http.Request) {
	urlSlice := strings.Split(r.URL.Path, "/")
	if len(urlSlice) < 3 {
		http.Error(w, "Bad request (url too short)", http.StatusBadRequest)
		return
	}

	heatUpMode := false
	if strings.ToLower(urlSlice[2]) == "on" {
		heatUpMode = true
	}
	
	for _, slave := range os.Sensors {
		if slave.Thermostat != nil {
			slave.Thermostat.HeatUpMode = heatUpMode
		}
	}
}

func (os *OwSet) StartServer() {

 	http.HandleFunc("/set", os.HandleSet)
 	http.HandleFunc("/heatup/", os.HandleAllHeatUp)
 	http.HandleFunc("/state", os.HandleState)
	
	go func(){
		log.Fatal(http.ListenAndServe(":80", nil))
	}()
}

type OwSlave struct {
	Name		string
	Id			uint64 
	Value		float64

	Thermostat		*Thermo
}

func (slave *OwSlave) SetFromInt(input uint64) {
	slave.Value = float64(input) / 1000
}

func (slave *OwSlave) InitThermo() {
	if slave.Thermostat == nil {
		return
	}

	if slave.Thermostat.Gpio == 0 {
		slave.Thermostat = nil
		log.Print("OwSlave InitThermo: thermostat found, but no gpio config - removing")
		return
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
		log.Printf("ERROR OwSlave InitThermo ReadState failed:\n%w", err)
	}

	log.Printf("OwSlave: Thermostat added!\n%v", slave.Thermostat)
}

func main() {
	log.Print("owkit started!")


	wires := OwSet{}
	wires.Set("./config.json")
	// log.Printf("%+v", wires)
	err := wires.InitSlaves()
	if err != nil {
		log.Fatal(err)
	}
	wires.PrintAll()

	log.Print("cycling..")
	wires.StartCycling()
	log.Print("starting http server..")
	wires.StartServer()

	for {
		time.Sleep(10 * time.Second)
	}
}