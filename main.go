package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	servicemaker "service-maker"
	"time"
)

func isCorrectFile(path string) bool {
	fileInfo, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return !fileInfo.IsDir()
}

var owkitService = servicemaker.ServiceMaker{
	User:               "owkit",
	UserGroups:         []string{},
	ServicePath:        "/etc/systemd/system/owkit.service",
	ServiceDescription: "Owkit service: reading 1-wire sensor temperature and thermostat with gpio rpi output.",
	ExecDir:            "/srv/owkit",
	ExecName:           "owkit",
	ExampleConfig: `{
		"Debug": true,
		"RefreshSeconds": 10,
		"Sensors": [{
			"Name": "sensor-name",
			"Id": 123456789,
			"Thermostat": {
				"Gpio": 21,
				"Hysteresis": 0.8,
				"Setpoint": 38,
				"HeatUp": 8
			}
		}],
		"LogInflux": {
			"Host": "http://localhost:8086/write",
			"Database": "dbname",
			"Measurment": "thermostat",
			"Tags": [{
				"Name": "tagname",
				"Value": "tagval"
			}]
		},
		"Server": {
			"Port": 8080,
			"IntMultiFactor": 2
		},
		"OffPeak": {
			"Url": "http://localhost:1234/offpeak"
		},
		"EnergyPanel": {
			"ConnectionString": "localhost:502",
			"HoldMinutes": 5,
			"PowerLevel": 1000
		}
	}`,
}

func main() {
	flagInstall := flag.Bool("install", false, "Install service in os")
	flag.Parse()
	if *flagInstall {
		err := owkitService.InstallService()
		if err != nil {
			panic(err)
		} else {
			fmt.Println("service installed!")
			return
		}
	}

	var err error
	var path string

	log.Print("owkit started!")

	confPaths := []string{"./config.json", "/etc/owkit.json"}
	for _, p := range confPaths {
		if isCorrectFile(p) {
			path = p
		}
	}
	if path == "" {
		log.Fatal("Config file not found!\n(looking in: %v)", confPaths)
	}

	wires := OwSet{}
	err = wires.Set(path)
	if err != nil {
		log.Fatal(err)
	}

	if wires.Debug {
		log.Println("Debugging is enabled!")
	}

	err = wires.InitSlaves()
	if err != nil {
		log.Fatal(err)
	}

	log.Print("cycling..")
	wires.StartCycling()

	if wires.Server != nil {
		log.Print("starting http server..")
		wires.Server.Start()
	}

	for {
		time.Sleep(10 * time.Second)
	}
}
