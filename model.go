package main

type PollenData struct {
	LastUpdate string   `json:"last_update,omitempty"`
	NextUpdate string   `json:"next_update,omitempty"`
	Sender     string   `json:"sender,omitempty"`
	Name       string   `json:"name,omitempty"`
	Legend     Legend   `json:"legend,omitempty"`
	Content    []Region `json:"content"`
}

type Legend struct {
	Id1      string `json:"id1,omitempty"`
	Id1_desc string `json:"id1_desc,omitempty"`
	Id2      string `json:"id2,omitempty"`
	Id2_desc string `json:"id2_desc,omitempty"`
	Id3      string `json:"id3,omitempty"`
	Id3_desc string `json:"id3_desc,omitempty"`
	Id4      string `json:"id4,omitempty"`
	Id4_desc string `json:"id4_desc,omitempty"`
	Id5      string `json:"id5,omitempty"`
	Id5_desc string `json:"id5_desc,omitempty"`
	Id6      string `json:"id6,omitempty"`
	Id6_desc string `json:"id6_desc,omitempty"`
	Id7      string `json:"id7,omitempty"`
	Id7_desc string `json:"id7_desc,omitempty"`
}

type Region struct {
	RegionId       int    `json:"region_id,omitempty"`
	PartregionId   int    `json:"partregion_id,omitempty"`
	RegionName     string `json:"region_name,omitempty"`
	PartregionName string `json:"partregion_name,omitempty"`
	Pollen         Pollen `json:"Pollen,omitempty"`
}

type PollenType struct {
	Today            string `json:"today,omitempty"`
	DayAfterTomorrow string `json:"dayafter_to,omitempty"`
	Tomorrow         string `json:"tomorrow,omitempty"`
}

type Pollen struct {
	Hasel    PollenType
	Erle     PollenType
	Esche    PollenType
	Birke    PollenType
	Graeser  PollenType
	Roggen   PollenType
	Beifuss  PollenType
	Ambrosia PollenType
}

// Hasel, Erle, Esche, Birke, Graeser, Roggen, Beifuss, Ambrosia

type HexalPollen struct {
	Name     string `xml:"name,attr"`
	Exposure string `xml:"belastung,attr"`
}

type HexalPollenExposure struct {
	Day    string        `xml:"tag,attr"`
	Pollen []HexalPollen `xml:"pollen"`
}

type HexalPollenData struct {
	Zip            string                `xml:"plz,attr"`
	City           string                `xml:"ort,attr"`
	PollenExposure []HexalPollenExposure `xml:"pollenbelastungen"`
}

type Datasets struct {
	Pollendaten []HexalPollenData `xml:"pollendaten"`
}
