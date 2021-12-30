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
	"time"
)

type apiResponse struct {
	Postcode    string
	Location    string
	Post_Office string
	State       string
}

type locations map[string]map[string]map[string]struct{}

type exportStruct struct {
	State     string
	Postcodes []struct {
		Postcode string
		Cities   []string
	}
}

const cachePath = ".pos-malaysia-postcode-scraper-cache"

func main() {
	start := flag.Int("start", 0, "starting point")
	end := flag.Int("end", 99999, "end point")
	step := flag.Int("step", 1, "loop step")
	interval := flag.Duration("interval", 0, "interval between fetches")
	out := flag.String("out", "all.json", "output file")
	ignoreSkipCache := flag.Bool("noSkipCache", false, "if set, will fetch previously found empty postcodes")
	flag.Parse()

	err := os.MkdirAll(cachePath, 0777)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	skips, err := getSkips()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	details := locations{}
	for i := *start; i <= *end; i = i + *step {
		pc := fmt.Sprintf("%05d", i)
		if _, ok := skips[pc]; !*ignoreSkipCache && ok {
			continue
		}

		fmt.Println("Fetching for", pc)
		u := fmt.Sprintf("https://api.pos.com.my/PostcodeWebApi/api/Postcode?Postcode=%s", pc)
		resp, err := http.Get(u)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		time.Sleep(*interval)

		ars := []apiResponse{}
		err = json.NewDecoder(resp.Body).Decode(&ars)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		if len(ars) == 0 {
			skips[pc] = struct{}{}
			err := saveSkips(skips)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			continue
		}

		for _, ar := range ars {
			if _, ok := details[ar.State]; !ok {
				details[ar.State] = make(map[string]map[string]struct{})
			}
			if _, ok := details[ar.State][ar.Postcode]; !ok {
				details[ar.State][ar.Postcode] = make(map[string]struct{})
			}
			details[ar.State][ar.Postcode][ar.Post_Office] = struct{}{}
		}
	}
	toExport := []exportStruct{}
	for state, ps := range details {
		s := exportStruct{State: state}
		for postcode, cs := range ps {
			p := struct {
				Postcode string
				Cities   []string
			}{Postcode: postcode}
			for city := range cs {
				p.Cities = append(p.Cities, city)
			}
			s.Postcodes = append(s.Postcodes, p)
		}
		toExport = append(toExport, s)
	}

	f, err := os.Create(*out)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer f.Close()
	err = json.NewEncoder(f).Encode(toExport)
	if err != nil {
		fmt.Println(err)
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
