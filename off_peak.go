package main 

import (
	"log"
	"net/http"
	"io/ioutil"
)

type OffPeak struct {
	Url		string
}
 
func (op *OffPeak) Check() bool {

	res, err := http.Get(op.Url)
	if err != nil {
		log.Printf("OffPeak Check, http get failed:\n%v\n", err)
		return false
	}
	info, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		log.Printf("OffPeak Check, body ReadAll failed:\n%v\n", err)
		return false
	}

	if string(info) == "true" {
		return true
	}

	return false
}