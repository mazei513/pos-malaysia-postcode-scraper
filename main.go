package main

import (
	"bytes"
	"encoding/json"
	"errors"
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

func (l locations) exists(postcode string) bool {
	for _, pcs := range l {
		_, ok := pcs[postcode]
		if ok {
			return true
		}
	}
	return false
}

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

const cachePath = ".pos-malaysia-postcode-scraper-cache"

func main() {
	start := flag.Int("start", 0, "starting point")
	end := flag.Int("end", 99999, "end point")
	step := flag.Int("step", 1, "loop step")
	interval := flag.Duration("interval", 0, "interval between fetches")
	out := flag.String("out", "all.json", "output file")
	ignoreSkipCache := flag.Bool("noSkipCache", false, "if set, will fetch previously found empty postcodes")
	ignoreCache := flag.Bool("noCache", false, "if set, will ignore previous responses")
	flag.Parse()

	err := os.MkdirAll(cachePath, 0777)
	if err != nil {
		fmt.Println("make cache folder: ", err)
		os.Exit(1)
	}

	skips, err := getSkips()
	if err != nil {
		fmt.Println("getSkips: ", err)
		os.Exit(1)
	}

	details := locations{}
	if !*ignoreCache {
		details, err = getCached()
		if err != nil {
			fmt.Println("getCached: ", err)
			os.Exit(1)
		}
	}

	for i := *start; i <= *end; i = i + *step {
		pc := fmt.Sprintf("%05d", i)
		if details.exists(pc) {
			fmt.Println("Found previous fetch for", pc)
			continue
		}
		if _, ok := skips[pc]; !*ignoreSkipCache && ok {
			continue
		}

		fmt.Println("Fetching for", pc)
		u := fmt.Sprintf("https://api.pos.com.my/PostcodeWebApi/api/Postcode?Postcode=%s", pc)
		resp, err := http.Get(u)
		if err != nil {
			fmt.Println("fetch response: ", err)
			os.Exit(1)
		}

		time.Sleep(*interval)

		ars := []apiResponse{}
		err = json.NewDecoder(resp.Body).Decode(&ars)
		if err != nil {
			fmt.Println("decode response", err)
			os.Exit(1)
		}

		if len(ars) == 0 {
			skips[pc] = struct{}{}
			err := saveSkips(skips)
			if err != nil {
				fmt.Println("saveSkips: ", err)
				os.Exit(1)
			}
			continue
		}

		for _, ar := range ars {
			details.store(ar)
		}
		err = saveCache(details)
		if err != nil {
			fmt.Println("saveCache: ", err)
			os.Exit(1)
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
}

func getSkips() (map[string]struct{}, error) {
	b, err := os.ReadFile(path.Join(cachePath, "skips"))
	res := map[string]struct{}{}
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return res, nil
		}
		return nil, err
	}
	for _, p := range bytes.Split(b, []byte(",")) {
		res[string(p)] = struct{}{}
	}
	return res, nil
}

func saveSkips(skips map[string]struct{}) error {
	b := [][]byte{}
	for s := range skips {
		b = append(b, []byte(s))
	}

	err := os.WriteFile(path.Join(cachePath, "skips"), bytes.Join(b, []byte(",")), 0666)
	if err != nil {
		return err
	}
	return nil
}

func getCached() (locations, error) {
	b, err := os.ReadFile(path.Join(cachePath, "cache.json"))
	res := locations{}
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return res, nil
		}
		return nil, err
	}
	err = json.Unmarshal(b, &res)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func saveCache(l locations) error {
	b, err := json.Marshal(l)
	if err != nil {
		return err
	}

	err = os.WriteFile(path.Join(cachePath, "cache.json"), b, 0666)
	if err != nil {
		return err
	}
	return nil
}
