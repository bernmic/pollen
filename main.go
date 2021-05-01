package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	POLLEN_URL = "http://opendata.dwd.de/climate_environment/health/alerts/s31fg.json"
	DATE_LAYOUT = "2006-01-02 15:04 Uhr"
)

var (
	cache Cache
)

type Cache struct {
	PollenData *PollenData
	LastUpdate *time.Time
	NextUpdate *time.Time
}

type RegionTemplateData struct {
	Region *Region
	Legend *Legend
}

func main() {
	cache = Cache{PollenData: nil, LastUpdate: nil, NextUpdate: nil}
	readPollenData()
	http.HandleFunc("/", handlerIndex)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handlerIndex(w http.ResponseWriter, r *http.Request) {
	if r.RequestURI == "" || r.RequestURI == "/" || r.RequestURI == "/index.html" {
		t, _ := template.ParseFiles("templates/index.html")
		pollenData, err := readPollenData()
		if err != nil {
			// todo output error
			fmt.Println(err)
		}
		t.Execute(w, pollenData)
	} else if strings.HasPrefix(r.RequestURI, "/region/") {
		handlerRegion(w, r)
	} else if _, err := os.Stat("assets/" + r.RequestURI); err == nil {
		serveFile(w, r)
	} else {
		renderNotFound(w, r)
	}
}

func handlerRegion(w http.ResponseWriter, r *http.Request) {
	u := r.RequestURI[8:]
	n := strings.Split(u, "/")
	if len(n) != 2 || n[1] == "" {
		renderNotFound(w, r)
		return
	}
	regionId, err := strconv.Atoi(n[0])
	if err != nil {
		renderNotFound(w, r)
		return
	}
	partregionId, err := strconv.Atoi(n[1])
	if err != nil {
		renderNotFound(w, r)
		return
	}
	p, err := readPollenData()
	if err != nil {
		renderServerError(w, r)
		return
	}
	for _, region := range p.Content {
		if region.RegionId == regionId && region.PartregionId == partregionId {
			t, err := template.ParseFiles("templates/region.html")
			if err != nil {
				renderServerError(w, r)
				return
			}
			t.Execute(w, region)
			return
		}
	}
	renderNotFound(w, r)
}

func readPollenData() (*PollenData, error) {
	if cache.NextUpdate != nil && cache.NextUpdate.After(time.Now()) {
		fmt.Println("Cache hit")
		return cache.PollenData, nil
	}

	pollenClient := http.Client{
		Timeout: time.Second * 5,
	}

	req, err := http.NewRequest(http.MethodGet, POLLEN_URL, nil)
	if err != nil {
		return nil, err
	}

	res, err := pollenClient.Do(req)
	if err != nil {
		return nil, err
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	pollenData := PollenData{}
	err = json.Unmarshal(body, &pollenData)
	if err != nil {
		return nil, err
	}
	updateCache(&pollenData)
	return &pollenData, nil
}

func updateCache(p *PollenData) {
	l, err := time.Parse(DATE_LAYOUT, p.LastUpdate)
	if err != nil {
		fmt.Println(err)
		return
	}
	n, err := time.Parse(DATE_LAYOUT, p.NextUpdate)
	if err != nil {
		fmt.Println(err)
		return
	}
	cache = Cache{PollenData: p, LastUpdate: &l, NextUpdate: &n}
}

func renderNotFound(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	fmt.Fprintf(w, "Could not find the page you requested: %s.", r.RequestURI)
}

func renderServerError(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusInternalServerError)
	fmt.Fprintf(w, "Internal Server Error: %s.", r.RequestURI)
}

func serveFile(w http.ResponseWriter, r *http.Request) {
	data, err := ioutil.ReadFile("assets/" + r.RequestURI)
	if err != nil {
		renderServerError(w, r)
		return
	}
	fmt.Println("Found file " + r.RequestURI)
	if strings.HasSuffix(r.RequestURI, ".css") {
		w.Header().Add("Content-Type", "text/css")
	}
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}
