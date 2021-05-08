package main

import (
	"encoding/xml"
	"fmt"
	"strconv"
	"testing"
)

func TestCreateXml(*testing.T) {
	d := Datasets{}
	d.Pollendaten = make([]HexalPollenData, 1)
	h := HexalPollenData{}
	h.Zip = "53797"
	h.City = "Lohmar"
	h.PollenExposure = make([]HexalPollenExposure, 7)

	for i := 0; i < 7; i++ {
		h.PollenExposure[i].Day = strconv.Itoa(i)
		h.PollenExposure[i].Pollen = make([]HexalPollen, 16)
		for j := 0; j < 16; j++ {
			h.PollenExposure[i].Pollen[j].Name = "FRITZ"
			h.PollenExposure[i].Pollen[j].Exposure = "0"
		}
	}

	d.Pollendaten[0] = h
	b, err := xml.Marshal(d)
	if err == nil {
		fmt.Println(string(b))
	}
}
