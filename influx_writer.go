package main

import (
	"fmt"
	"bytes"
	"net/http"
)

type InfluxWriter struct {
	Host			string
	Database		string
	Measurment		string

	ForceUseId		bool		`json:",omitempty"`

	Tags			[]Tag		`json:",omitempty"`
}

type Tag struct {
	Name	string
	Value	string
}

func (ifw *InfluxWriter) Send(slaves []*OwSlave) (error) {
	var query string

	for _, slave := range slaves {
		query += ifw.GetLine(slave)
	}

	req, err := http.NewRequest("POST", ifw.Host + "?db=" + ifw.Database, bytes.NewBufferString(query))
	if err != nil {
        return fmt.Errorf("InfluxWriter Send: preparing request error:\n%w", err)
    }
	client := &http.Client{}
    resp, err2 := client.Do(req)
    if err2 != nil {
        return fmt.Errorf("InfluxWriter Send: client.Do error:\n%w", err2)
    }
    defer resp.Body.Close()
    if resp.StatusCode / 100 != 2 {
    	return fmt.Errorf("InfluxWriter Send: received non-success response code: %d", resp.StatusCode)
    }
    
    return nil 
}

func (ifw *InfluxWriter) GetLine(slave *OwSlave) (line string) {
	line = ifw.Measurment

	idTag := Tag{Name: "id"}
	if len(slave.Name) > 0 {
		idTag.Value = slave.Name
	} else {
		idTag.Value = fmt.Sprintf("%012x", slave.Id)
	}

	tags := append(ifw.Tags, idTag)

	for _, tag := range tags {
		line += fmt.Sprintf(",%s=%s", tag.Name, tag.Value)
	}

	line += fmt.Sprintf(" temperature=%f\n", slave.Value)

	return
}
