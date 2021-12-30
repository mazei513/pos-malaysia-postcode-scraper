package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
)

type exporter struct {
	f   *os.File
	enc *json.Encoder
}

func newExporterToPath(path string) (exporter, error) {
	f, err := os.Create(path)
	if err != nil {
		return exporter{}, fmt.Errorf("failed to create export file: %w", err)
	}

	enc := json.NewEncoder(f)
	enc.SetEscapeHTML(false)

	return exporter{f, enc}, nil
}

func (e exporter) Close() error {
	return e.f.Close()
}

func (e exporter) exportLocations(l locations) error {
	toExport := []exportState{}
	for state, ps := range l {
		s := exportState{State: state}
		for postcode, cs := range ps {
			p := exportPostCode{Postcode: postcode}
			for city, ls := range cs {
				c := exportCity{City: city}
				for l := range ls {
					c.Locations = append(c.Locations, l)
				}
				sort.Strings(c.Locations)
				p.Cities = append(p.Cities, c)
			}
			sort.SliceStable(p.Cities, func(i, j int) bool { return p.Cities[i].City < p.Cities[j].City })
			s.Postcodes = append(s.Postcodes, p)
		}
		sort.SliceStable(s.Postcodes, func(i, j int) bool { return s.Postcodes[i].Postcode < s.Postcodes[j].Postcode })
		toExport = append(toExport, s)
	}
	sort.SliceStable(toExport, func(i, j int) bool { return toExport[i].State < toExport[j].State })
	return e.enc.Encode(toExport)
}
