package main

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go"
	"github.com/influxdata/influxdb-client-go/api/write"
)

type InfluxWriter struct {
	Host         string
	Database     string
	Measurment   string
	Organization string
	Token        string

	ForceUseId bool `json:",omitempty"`
	UseInflux1 bool

	Tags []Tag `json:",omitempty"`
}

type Tag struct {
	Name  string
	Value string
}

func getTagMap(tags []Tag) (tagMap map[string]string) {
	tagMap = map[string]string{}
	for _, tag := range tags {
		tagMap[tag.Name] = tag.Value
	}

	return
}

func (ifw *InfluxWriter) Send(slaves []*OwSlave) error {
	if ifw.UseInflux1 {
		return ifw.SendWithInflux1(slaves)
	}

	client := influxdb2.NewClient(ifw.Host, ifw.Token)
	writeAPI := client.WriteAPIBlocking(ifw.Organization, ifw.Database)
	defer client.Close()
	var slavePoint, thermoPoint *write.Point
	var err error
	for _, slave := range slaves {
		tags := getTagMap(append(ifw.Tags, ifw.getIdTag(slave)))
		slavePoint = influxdb2.NewPoint(ifw.Measurment,
			tags,
			map[string]interface{}{"temperature": slave.Value},
			time.Now())
		if slave.Thermostat != nil {
			thermoPoint = influxdb2.NewPoint(ifw.Measurment,
				tags,
				map[string]interface{}{
					"setpoint": slave.Thermostat.Setpoint,
					"real-sp":  slave.Thermostat.GetSetpoint(),
					"state":    slave.Thermostat.CheckIfOn(),
					"heatup":   slave.Thermostat.CheckIfHeatUp(),
				},
				time.Now())
			err = writeAPI.WritePoint(context.Background(), thermoPoint)
			if err != nil {
				return err
			}
		}
		err = writeAPI.WritePoint(context.Background(), slavePoint)
		if err != nil {
			return err
		}
	}
	return nil
}

func (ifw *InfluxWriter) SendWithInflux1(slaves []*OwSlave) error {
	var query string

	for _, slave := range slaves {
		query += ifw.GetLine(slave)
		if slave.Thermostat != nil {
			query += ifw.GetThermoLines(slave.Thermostat)
		}
	}

	req, err := http.NewRequest("POST", ifw.Host+"?db="+ifw.Database, bytes.NewBufferString(query))
	if err != nil {
		return fmt.Errorf("InfluxWriter Send: preparing request error:\n%w", err)
	}
	client := &http.Client{}
	resp, err2 := client.Do(req)
	if err2 != nil {
		return fmt.Errorf("InfluxWriter Send: client.Do error:\n%w", err2)
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("InfluxWriter Send: received non-success response code: %d", resp.StatusCode)
	}

	return nil
}

func (ifw *InfluxWriter) getIdTag(slave *OwSlave) (idTag Tag) {
	idTag = Tag{Name: "id"}
	if len(slave.Name) > 0 {
		idTag.Value = slave.Name
	} else {
		idTag.Value = fmt.Sprintf("%012x", slave.Id)
	}
	return
}

func (ifw *InfluxWriter) GetLine(slave *OwSlave) (line string) {
	line = ifw.Measurment

	tags := append(ifw.Tags, ifw.getIdTag(slave))

	for _, tag := range tags {
		line += fmt.Sprintf(",%s=%s", tag.Name, tag.Value)
	}

	line += fmt.Sprintf(" temperature=%f\n", slave.Value)

	return
}

func (ifw *InfluxWriter) GetThermoLines(thermo *Thermo) (line string) {
	baseline := ifw.Measurment

	tags := append(ifw.Tags, ifw.getIdTag(thermo.Sensor))
	for _, tag := range tags {
		baseline += fmt.Sprintf(",%s=%s", tag.Name, tag.Value)
	}

	line = baseline + fmt.Sprintf(" setpoint=%f\n", thermo.Setpoint)
	line += baseline + fmt.Sprintf(" real-sp=%f\n", thermo.GetSetpoint())
	line += baseline + fmt.Sprintf(" state=%v\n", thermo.CheckIfOn())
	line += baseline + fmt.Sprintf(" heatup=%v\n", thermo.CheckIfHeatUp())

	return
}
