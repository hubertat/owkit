package main

import (
	"fmt"
	"bytes"
	"time"
	"math"
	"net/http"
	"net/url"
	"encoding/json"
)


type HttpWriter struct {
	Host,
	Method		string

	ForceUseId		bool		`json:",omitempty"`
	IntMultiFactor	int
}

func (hw *HttpWriter) Send(slaves []*OwSlave) error {
	var req *http.Request
	var err error

	switch hw.Method {
	case http.MethodGet:
		url, err := url.Parse(hw.Host)
		if err != nil {
			return fmt.Errorf("HttpWriter Send parsing host url (%v) failed:\n%v", hw.Host, err)
		}
		if url.Scheme == "" {
			url.Scheme = "http"
		}
		url.RawQuery = hw.getSlavesQuery(slaves).Encode()
		req, err = http.NewRequest(hw.Method, url.String(), nil)
		if err != nil {
			return fmt.Errorf("HttpWriter Send NewRequest (%v) failed:\n%v", hw.Method, err)
		}
	case http.MethodPost:
		json, err := json.Marshal(slaves)
		if err != nil {
			return fmt.Errorf("HttpWriter Send json Marshal failed:\n%v", err)
		}
		req, err = http.NewRequest(hw.Method, hw.Host, bytes.NewReader(json))
		if err != nil {
			return fmt.Errorf("HttpWriter Send NewRequest (%v) failed:\n%v", hw.Method, err)
		}		
	default:
		return fmt.Errorf("HttpWriter Send: missing or unsupported http method\n")
	}


	req.Header.Set("Content-Type", "application/json")

	client := http.Client{
		Timeout: 6 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("HttpWriter Send http Client failed:\n%v", err)
	}

	defer resp.Body.Close()
	if resp.StatusCode >= 300 || resp.StatusCode < 200 {
		return fmt.Errorf("HttpWriter Send error:\nReceived non-success http response from grenton host:\n%v", resp.Status)
	}

	return nil
}

func (hw *HttpWriter) getSlavesQuery(slaves []*OwSlave) (query url.Values) {
	query = url.Values{}

	for _, slv := range slaves {
		query.Add(slv.Name, fmt.Sprintf("%.0f", math.Round(slv.Value * math.Pow10(hw.IntMultiFactor))))	
	}

	return
}