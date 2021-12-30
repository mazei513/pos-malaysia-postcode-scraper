package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path"
	"strings"
	"time"
)

type apiResponse struct {
	Postcode    string
	Location    string
	Post_Office string
	State       string
}

type locations map[string]map[string]map[string]map[string]struct{}

func (l locations) store(ar apiResponse) {
	if _, ok := l[ar.State]; !ok {
		l[ar.State] = make(map[string]map[string]map[string]struct{})
	}
	if _, ok := l[ar.State][ar.Postcode]; !ok {
		l[ar.State][ar.Postcode] = make(map[string]map[string]struct{})
	}
	if _, ok := l[ar.State][ar.Postcode][ar.Post_Office]; !ok {
		l[ar.State][ar.Postcode][ar.Post_Office] = make(map[string]struct{})
	}
	l[ar.State][ar.Postcode][ar.Post_Office][strings.TrimSpace(ar.Location)] = struct{}{}
}

type exportCity struct {
	City      string   `json:"city"`
	Locations []string `json:"locations"`
}

type exportPostCode struct {
	Postcode string       `json:"postcode"`
	Cities   []exportCity `json:"cities"`
}

type exportState struct {
	State     string           `json:"state"`
	Postcodes []exportPostCode `json:"postcodes"`
}

type logger struct{ quiet bool }

func (l logger) println(a ...interface{}) {
	if l.quiet {
		return
	}
	fmt.Println(a...)
}

const cachePath = ".pos-malaysia-postcode-scraper-cache"

func main() {
	start := flag.Int("start", 0, "starting point")
	end := flag.Int("end", 99999, "end point")
	step := flag.Int("step", 1, "loop step")
	interval := flag.Duration("interval", 0, "interval between fetches")
	out := flag.String("out", "all.json", "output file")
	ignoreCache := flag.Bool("noCache", false, "if set, will ignore previous responses")
	quiet := flag.Bool("q", false, "if set, will not print to stdout")
	flag.Parse()

	log := logger{*quiet}

	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("home dir: ", err)
		os.Exit(1)
	}

	cache := cacheManager{path.Join(home, cachePath)}

	err = cache.init()
	if err != nil {
		fmt.Println("init cache: ", err)
		os.Exit(1)
	}

	details := locations{}
	for i := *start; i <= *end; i = i + *step {
		pc := fmt.Sprintf("%05d", i)

		ars, err := cache.get(pc)
		if *ignoreCache || err != nil {
			log.println("Fetching for", pc)
			u := fmt.Sprintf("https://api.pos.com.my/PostcodeWebApi/api/Postcode?Postcode=%s", pc)
			resp, err := http.Get(u)
			if err != nil {
				fmt.Println("fetch response: ", err)
				os.Exit(1)
			}

			time.Sleep(*interval)

			ars = []apiResponse{}
			err = json.NewDecoder(resp.Body).Decode(&ars)
			if err != nil {
				fmt.Println("decode response", err)
				os.Exit(1)
			}

			err = cache.set(pc, ars)
			if err != nil {
				fmt.Println("cacheResponse: ", err)
				os.Exit(1)
			}
		} else {
			log.println("Found cached response for", pc)
		}

		for _, ar := range ars {
			details.store(ar)
		}
	}

	toExport := []exportState{}
	for state, ps := range details {
		s := exportState{State: state}
		for postcode, cs := range ps {
			p := exportPostCode{Postcode: postcode}
			for city, ls := range cs {
				c := exportCity{City: city}
				for l := range ls {
					c.Locations = append(c.Locations, l)
				}
				p.Cities = append(p.Cities, c)
			}
			s.Postcodes = append(s.Postcodes, p)
		}
		toExport = append(toExport, s)
	}

	f, err := os.Create(*out)
	if err != nil {
		fmt.Println("create output file: ", err)
		os.Exit(1)
	}
	defer f.Close()
	err = json.NewEncoder(f).Encode(toExport)
	if err != nil {
		fmt.Println("write output file:", err)
		os.Exit(1)
	}
	log.println("stored in", *out)
}

type cacheManager struct {
	path string
}

func (c cacheManager) init() error {
	return os.MkdirAll(c.path, 0777)
}

func (c cacheManager) get(postcode string) ([]apiResponse, error) {
	b, err := os.ReadFile(path.Join(c.path, postcode))
	if err != nil {
		return nil, err
	}
	res := []apiResponse{}
	err = json.Unmarshal(b, &res)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (c cacheManager) set(postcode string, a []apiResponse) error {
	b, err := json.Marshal(a)
	if err != nil {
		return err
	}

	err = os.WriteFile(path.Join(c.path, postcode), b, 0666)
	if err != nil {
		return err
	}
	return nil
}
