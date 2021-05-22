package main

import (
	"embed"
	"encoding/json"
	"encoding/xml"
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
	REGION_URI    = "/region/"
	ZIP_URI       = "/zip/"
	HEXAL_URL     = "http://www.allergie.hexal.de/pollenflug/xml-interface-neu/pollen_de_7tage.php?plz="
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

type HexalTemplateData struct {
	Zip      string
	City     string
	Dates    []time.Time
	Ambrosia []string
	Ampfer   []string
	Beifuss  []string
	Birke    []string
	Buche    []string
	Eiche    []string
	Erle     []string
	Esche    []string
	Graeser  []string
	Hasel    []string
	Pappel   []string
	Roggen   []string
	Ulme     []string
	Wegerich []string
	Weide    []string
}

//go:embed templates
var templates embed.FS

//go:embed assets
var assets embed.FS

func main() {
	log.Println("Starting pollen.")
	if ps, ok := os.LookupEnv(ENV_PORT); ok {
		p, err := strconv.Atoi(ps)
		if err == nil {
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
	if r.RequestURI == "" || r.RequestURI == "/" {
		t, _ := template.ParseFS(templates, templateDir+"/index.html")
		pollenData, err := readPollenData()
		if err != nil {
			accessLog(r, http.StatusInternalServerError, err.Error())
			renderServerError(w, r)
			return
		}
		t.Execute(w, pollenData)
		accessLog(r, http.StatusOK, "")
	} else if r.RequestURI == "/index.html" {
		http.Redirect(w, r, "./", http.StatusMovedPermanently)
		accessLog(r, http.StatusMovedPermanently, "Redirect to '/'")
	} else if strings.HasPrefix(r.RequestURI, "/region/") {
		handlerRegion(w, r)
	} else if strings.HasPrefix(r.RequestURI, "/zip") {
		handlerZip(w, r)
	} else if f, err := assets.Open(assetsDir + r.RequestURI); err == nil {
		f.Close()
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
			t, _ := template.ParseFS(templates, templateDir+"/region.html")
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

func handlerZip(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			fmt.Fprintf(w, "ParseForm() err: %v", err)
			return
		}
		zip := r.FormValue("zip")

		http.Redirect(w, r, r.Referer()+"zip"+"/"+zip, http.StatusMovedPermanently)
		accessLog(r, http.StatusMovedPermanently, "Redirect to /zip/"+zip)
		return
	}
	u := r.RequestURI[5:]
	p, err := readHexal(u)
	if err != nil {
		renderServerError(w, r)
	}
	t := HexalTemplateData{}
	t.Ambrosia = make([]string, 7)
	t.Ampfer = make([]string, 7)
	t.Beifuss = make([]string, 7)
	t.Birke = make([]string, 7)
	t.Buche = make([]string, 7)
	t.Eiche = make([]string, 7)
	t.Erle = make([]string, 7)
	t.Esche = make([]string, 7)
	t.Graeser = make([]string, 7)
	t.Hasel = make([]string, 7)
	t.Pappel = make([]string, 7)
	t.Roggen = make([]string, 7)
	t.Ulme = make([]string, 7)
	t.Wegerich = make([]string, 7)
	t.Weide = make([]string, 7)
	t.Dates = make([]time.Time, 7)
	w.WriteHeader(200)
	for _, x := range p.Pollendaten {
		t.Zip = x.Zip
		t.City = x.City
		for _, y := range x.PollenExposure {
			day, err := strconv.Atoi(y.Day)
			if err != nil || day > 6 {
				renderServerError(w, r)
				return
			}
			t.Dates[day] = time.Now().AddDate(0, 0, day)
			for _, z := range y.Pollen {
				switch z.Name {
				case "Ambrosia":
					t.Ambrosia[day] = z.Exposure
				case "Ampfer":
					t.Ampfer[day] = z.Exposure
				case "Beifuß":
					t.Beifuss[day] = z.Exposure
				case "Birke":
					t.Birke[day] = z.Exposure
				case "Buche":
					t.Buche[day] = z.Exposure
				case "Eiche":
					t.Eiche[day] = z.Exposure
				case "Erle":
					t.Erle[day] = z.Exposure
				case "Esche":
					t.Esche[day] = z.Exposure
				case "Gräser":
					t.Graeser[day] = z.Exposure
				case "Hasel":
					t.Hasel[day] = z.Exposure
				case "Pappel":
					t.Pappel[day] = z.Exposure
				case "Roggen":
					t.Roggen[day] = z.Exposure
				case "Ulme":
					t.Ulme[day] = z.Exposure
				case "Wegerich":
					t.Wegerich[day] = z.Exposure
				case "Weide":
					t.Weide[day] = z.Exposure
				}
			}
		}
	}
	template, err := template.ParseFS(templates, templateDir+"/zip.html")
	if err != nil {
		accessLog(r, http.StatusInternalServerError, err.Error())
		renderServerError(w, r)
		return
	}
	template.Execute(w, t)
	accessLog(r, http.StatusOK, "")
}

func readHexal(zip string) (*Datasets, error) {
	pollenClient := http.Client{
		Timeout: time.Second * 5,
	}

	req, err := http.NewRequest(http.MethodGet, HEXAL_URL+zip, nil)
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
	pollenData := Datasets{}
	err = xml.Unmarshal(body, &pollenData)
	if err != nil {
		return nil, err
	}
	return &pollenData, nil
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
	data, err := assets.ReadFile(assetsDir + r.RequestURI)
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
	case strings.HasSuffix(lc, ".map"):
		w.Header().Add("Content-Type", "application/json")
	case strings.HasSuffix(lc, ".svg"):
		w.Header().Add("Content-Type", "image/svg+xml")
	}
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func accessLog(r *http.Request, httpCode int, payload string) {
	log.Printf("%s %s, %d, %s", r.Method, r.RequestURI, httpCode, payload)
}

func getDates() (today time.Time, tomorrow time.Time, dayAfterTomorrow time.Time) {
	t := cache.LastUpdate
	today = *t
	tomorrow = t.AddDate(0, 0, 1)
	dayAfterTomorrow = t.AddDate(0, 0, 2)
	return
}
