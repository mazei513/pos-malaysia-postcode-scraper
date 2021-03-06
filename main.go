package main

import (
	"flag"
	"fmt"
	"os"
	"path"
	"strings"
)

type apiResponse struct {
	Postcode    string
	Location    string
	Post_Office string
	State       string
}

type locations map[string]map[string]map[string]map[string]struct{}

func (l locations) store(ar apiResponse) {
	state := strings.TrimSpace(ar.State)
	city := strings.TrimSpace(ar.Post_Office)
	postcode := strings.TrimSpace(ar.Postcode)
	location := strings.TrimSpace(ar.Location)
	if _, ok := l[state]; !ok {
		l[state] = make(map[string]map[string]map[string]struct{})
	}
	if _, ok := l[state][city]; !ok {
		l[state][city] = make(map[string]map[string]struct{})
	}
	if _, ok := l[state][city][postcode]; !ok {
		l[state][city][postcode] = make(map[string]struct{})
	}
	l[state][city][postcode][location] = struct{}{}
}

func main() {
	start := flag.Int("start", 0, "starting point")
	end := flag.Int("end", 99999, "end point")
	step := flag.Int("step", 1, "loop step")
	interval := flag.Duration("interval", 0, "interval between fetches")
	out := flag.String("out", "all.json", "output file")
	ignoreCache := flag.Bool("noCache", false, "if set, will ignore previous responses")
	quiet := flag.Bool("q", false, "if set, will not print to stdout")
	excludeLocations := flag.Bool("noLocations", false, "if set, will exclude locations field")

	flag.Parse()

	log := logger{*quiet}

	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("home dir: ", err)
		os.Exit(1)
	}

	cache := cacheManager{path: path.Join(home, ".pos-malaysia-postcode-scraper-cache")}
	err = cache.makeFolder()
	if err != nil {
		fmt.Println("init cache: ", err)
		os.Exit(1)
	}

	f := fetcher{cache: cache, log: log, interval: *interval, ignoreCache: *ignoreCache}

	details := locations{}
	for i := *start; i <= *end; i = i + *step {
		pc := fmt.Sprintf("%05d", i)

		ars, err := f.get(pc)
		if err != nil {
			fmt.Println("fetcher get: ", err)
			os.Exit(1)
		}

		for _, ar := range ars {
			details.store(ar)
		}
	}

	exp, err := newExporterToPath(*out)
	if err != nil {
		fmt.Println("create output file: ", err)
		os.Exit(1)
	}
	defer exp.Close()

	if *excludeLocations {
		err = exp.exportWithoutLocations(details)
	} else {
		err = exp.export(details)
	}
	if err != nil {
		fmt.Println("write output file:", err)
		os.Exit(1)
	}
	log.println("stored in", *out)
}
