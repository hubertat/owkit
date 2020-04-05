package main

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"math"
	"encoding/json"
)

type Server struct {
	Port				uint
	IntMultiFactor		int

	set					*OwSet
}

func (srv *Server) HandleSet(w http.ResponseWriter, r *http.Request) {
	setS := &OwSlave{}
    err := json.NewDecoder(r.Body).Decode(setS)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    for _, slave := range srv.set.Sensors {
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

func (srv *Server) HandleState(w http.ResponseWriter, r *http.Request) {
	
	js, err := json.Marshal(srv.set)
	
	if err != nil {
	http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func (srv *Server) HandleAllHeatUp(w http.ResponseWriter, r *http.Request) {
	urlSlice := strings.Split(r.URL.Path, "/")
	if len(urlSlice) < 3 {
		http.Error(w, "Bad request (url too short)", http.StatusBadRequest)
		return
	}

	heatUpMode := false
	if strings.ToLower(urlSlice[2]) == "on" {
		heatUpMode = true
	}
	
	for _, slave := range srv.set.Sensors {
		if slave.Thermostat != nil {
			slave.Thermostat.HeatUpMode = heatUpMode
		}
	}
}

func (srv *Server) HandleSetpointIncrease(w http.ResponseWriter, r *http.Request) {
	urlSlice := strings.Split(r.URL.Path, "/")

	var slaves []*OwSlave

	if len(urlSlice) > 2 {
		slaves = append(slaves, srv.set.GetSlave(urlSlice[2]))
	} else {
		slaves = srv.set.Sensors
	}

	for _, slave := range slaves {
		slave.Thermostat.Setpoint = slave.Thermostat.Setpoint + 0.5
	}
}

func (srv *Server) HandleSetpointDecrease(w http.ResponseWriter, r *http.Request) {
	urlSlice := strings.Split(r.URL.Path, "/")

	var slaves []*OwSlave

	if len(urlSlice) > 2 {
		slaves = append(slaves, srv.set.GetSlave(urlSlice[2]))
	} else {
		slaves = srv.set.Sensors
	}

	for _, slave := range slaves {
		slave.Thermostat.Setpoint = slave.Thermostat.Setpoint - 0.5
	}
}

func (srv *Server) HandleSetSetpoint(w http.ResponseWriter, r *http.Request) {
	urlSlice := strings.Split(r.URL.Path, "/")
	if len(urlSlice) < 4 {
		http.Error(w, "Bad request (url too short)", http.StatusBadRequest)
		return
	}

	var slave *OwSlave
	intId, err := strconv.ParseUint(urlSlice[2], 10, 64)
	if err != nil {
		slave = srv.set.GetSlaveById(intId)
	}
	if slave == nil {
		slave = srv.set.GetSlaveByName(urlSlice[2])
	}
	if slave == nil {
		http.Error(w, "Slave sensor not found", 404)
		return
	}

	spInt, err := strconv.ParseInt(urlSlice[3], 10, 64)
	if err != nil {
		http.Error(w, "Error during parsing sepoint value", http.StatusBadRequest)
		return
	}

	if slave.Thermostat == nil {
		http.Error(w, "Selected slave sensor doesnt have thermostat", 404)
		return
	}
	
	slave.Thermostat.Setpoint = float64(spInt) / math.Pow10(srv.IntMultiFactor)
}


func (srv *Server) Start() {

 	http.HandleFunc("/set", srv.HandleSet)
 	http.HandleFunc("/setpoint/", srv.HandleSetSetpoint)
 	http.HandleFunc("/increase/", srv.HandleSetpointIncrease)
 	http.HandleFunc("/decrease/", srv.HandleSetpointDecrease)
 	http.HandleFunc("/heatup/", srv.HandleAllHeatUp)
 	http.HandleFunc("/state", srv.HandleState)
	
	go func(){
		fmt.Println(http.ListenAndServe(fmt.Sprintf(":%d", srv.Port), nil))
	}()
}
