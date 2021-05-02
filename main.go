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
	ENV_PORT      = "POLLEN_PORT"
	ENV_ASSETS    = "POLLEN_ASSETS_DIR"
	ENV_TEMPLATES = "POLLEN_TEMPLATES_DIR"
	POLLEN_URL    = "http://opendata.dwd.de/climate_environment/health/alerts/s31fg.json"
	DATE_LAYOUT   = "2006-01-02 15:04 Uhr"
)

var (
	cache       Cache
	port        = 8080
	templateDir = "templates"
	assetsDir   = "assets"
)

type Cache struct {
	PollenData *PollenData
	LastUpdate *time.Time
	NextUpdate *time.Time
}
type RegionTemplateData struct {
	*Region
	LastUpdate       *time.Time
	NextUpdate       *time.Time
	Today            *time.Time
	Tomorrow         *time.Time
	DayAfterTomorrow *time.Time
}

func main() {
	log.Println("Starting pollen.")
	if ps, ok := os.LookupEnv(ENV_PORT); ok {
		p, err := strconv.Atoi(ps)
		if err != nil {
			port = p
		}
	}
	if ts, ok := os.LookupEnv(ENV_TEMPLATES); ok {
		templateDir = ts
	}
	if as, ok := os.LookupEnv(ENV_ASSETS); ok {
		assetsDir = as
	}

	log.Printf("Using port %d, templates %s and assets %s", port, templateDir, assetsDir)

	cache = Cache{PollenData: nil, LastUpdate: nil, NextUpdate: nil}
	readPollenData()
	http.HandleFunc("/", handlerIndex)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%04d", port), nil))
}

func handlerIndex(w http.ResponseWriter, r *http.Request) {
	if r.RequestURI == "" || r.RequestURI == "/" || r.RequestURI == "/index.html" {
		t, _ := template.ParseFiles(templateDir + "/index.html")
		pollenData, err := readPollenData()
		if err != nil {
			accessLog(r, http.StatusInternalServerError, err.Error())
			renderServerError(w, r)
			return
		}
		t.Execute(w, pollenData)
		accessLog(r, http.StatusOK, "")
	} else if strings.HasPrefix(r.RequestURI, "/region/") {
		handlerRegion(w, r)
	} else if _, err := os.Stat(assetsDir + "/" + r.RequestURI); err == nil {
		serveFile(w, r)
	} else {
		renderNotFound(w, r)
		accessLog(r, http.StatusNotFound, "")
	}
}

func handlerRegion(w http.ResponseWriter, r *http.Request) {
	u := r.RequestURI[8:]
	n := strings.Split(u, "/")
	if len(n) != 2 || n[1] == "" {
		accessLog(r, http.StatusNotFound, "Invalid region / partregion")
		renderNotFound(w, r)
		return
	}
	regionId, err := strconv.Atoi(n[0])
	if err != nil {
		accessLog(r, http.StatusNotFound, err.Error())
		renderNotFound(w, r)
		return
	}
	partregionId, err := strconv.Atoi(n[1])
	if err != nil {
		accessLog(r, http.StatusNotFound, err.Error())
		renderNotFound(w, r)
		return
	}
	p, err := readPollenData()
	if err != nil {
		accessLog(r, http.StatusInternalServerError, err.Error())
		renderServerError(w, r)
		return
	}
	for _, region := range p.Content {
		if region.RegionId == regionId && region.PartregionId == partregionId {
			t, err := template.ParseFiles(templateDir + "/region.html")
			if err != nil {
				accessLog(r, http.StatusInternalServerError, err.Error())
				renderServerError(w, r)
				return
			}
			today, tomorrow, dayAfterTomorrow := getDates()
			regionTemplateData := RegionTemplateData{&region, cache.LastUpdate, cache.NextUpdate, &today, &tomorrow, &dayAfterTomorrow}
			t.Execute(w, regionTemplateData)
			accessLog(r, http.StatusOK, "")
			return
		}
	}
	accessLog(r, http.StatusNotFound, "Unknown region / partregion")
	renderNotFound(w, r)
}

func readPollenData() (*PollenData, error) {
	if cache.NextUpdate != nil && cache.NextUpdate.After(time.Now()) {
		log.Println("Serve pollen data from cache")
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
	log.Println("New pollen data loaded from DWD")
	log.Printf("Pollen data was updated %s. Next update is %s.", cache.LastUpdate.Format("2006-01-02 15:04"), cache.NextUpdate.Format("2006-01-02 15:04"))
	return &pollenData, nil
}

func updateCache(p *PollenData) {
	l, err := time.Parse(DATE_LAYOUT, p.LastUpdate)
	if err != nil {
		log.Println(err)
		return
	}
	n, err := time.Parse(DATE_LAYOUT, p.NextUpdate)
	if err != nil {
		log.Println(err)
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
	data, err := ioutil.ReadFile(assetsDir + "/" + r.RequestURI)
	if err != nil {
		accessLog(r, http.StatusInternalServerError, err.Error())
		renderServerError(w, r)
		return
	}
	accessLog(r, 200, "")
	lc := strings.ToLower(r.RequestURI)
	switch {
	case strings.HasSuffix(lc, ".css"):
		w.Header().Add("Content-Type", "text/css")
	case strings.HasSuffix(lc, ".jpg"):
		w.Header().Add("Content-Type", "image/jpeg")
	case strings.HasSuffix(lc, ".jpeg"):
		w.Header().Add("Content-Type", "image/jpeg")
	case strings.HasSuffix(lc, ".png"):
		w.Header().Add("Content-Type", "image/png")
	case strings.HasSuffix(lc, ".gif"):
		w.Header().Add("Content-Type", "image/git")
	case strings.HasSuffix(lc, ".ico"):
		w.Header().Add("Content-Type", "image/x-icon")
	case strings.HasSuffix(lc, ".html"):
		w.Header().Add("Content-Type", "text/html")
	case strings.HasSuffix(lc, ".js"):
		w.Header().Add("Content-Type", "application/javascript")
	case strings.HasSuffix(lc, ".svg"):
		w.Header().Add("Content-Type", "image/svg+xml")
	}
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func accessLog(r *http.Request, httpCode int, payload string) {
	log.Printf("%s, %d, %s", r.RequestURI, httpCode, payload)
}

func getDates() (today time.Time, tomorrow time.Time, dayAfterTomorrow time.Time) {
	t := cache.LastUpdate
	today = *t
	tomorrow = t.AddDate(0, 0, 1)
	dayAfterTomorrow = t.AddDate(0, 0, 2)
	return
}
