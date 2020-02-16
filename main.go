package main

import (
	"log"
	"time"
	"os"
)

func isCorrectFile(path string) bool {
	fileInfo, err := os.Stat(path)
    if os.IsNotExist(err) {
        return false
    }
    return !fileInfo.IsDir()
}

func main() {
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

	err = wires.InitSlaves()
	if err != nil {
		log.Fatal(err)
	}

	
	log.Print("cycling..")
	wires.StartCycling()
	
	log.Print("starting http server..")
	wires.Server.Start()

	for {
		time.Sleep(10 * time.Second)
	}
}