package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
)

type exportPostCode struct {
	Postcode  string   `json:"postcode"`
	City      string   `json:"city"`
	Locations []string `json:"locations"`
}

type exportState struct {
	State     string           `json:"state"`
	Postcodes []exportPostCode `json:"postcodes,omitempty"`
}

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

func (e exporter) export(l locations) error {
	toExport := []exportState{}
	for state, cities := range l {
		s := exportState{State: state}
		for city, postcodes := range cities {
			for postcode, locations := range postcodes {
				p := exportPostCode{Postcode: postcode, City: city}
				for location := range locations {
					p.Locations = append(p.Locations, location)
				}
				sort.Strings(p.Locations)
				s.Postcodes = append(s.Postcodes, p)
			}
		}
		sort.SliceStable(s.Postcodes, func(i, j int) bool {
			return s.Postcodes[i].Postcode < s.Postcodes[j].Postcode || (s.Postcodes[i].Postcode == s.Postcodes[j].Postcode && s.Postcodes[i].City < s.Postcodes[j].City)
		})
		toExport = append(toExport, s)
	}
	sort.SliceStable(toExport, func(i, j int) bool { return toExport[i].State < toExport[j].State })
	return e.enc.Encode(toExport)
}

type exportNoLocationsPostcode struct {
	Postcode string `json:"postcode"`
	City     string `json:"city"`
}

type exportNoLocations struct {
	State     string                      `json:"state"`
	Postcodes []exportNoLocationsPostcode `json:"postcodes"`
}

func (e exporter) exportWithoutLocations(l locations) error {
	toExport := []exportNoLocations{}
	for state, cs := range l {
		s := exportNoLocations{State: state}
		for city, ps := range cs {
			for postcode := range ps {
				s.Postcodes = append(s.Postcodes, exportNoLocationsPostcode{Postcode: postcode, City: city})
			}
		}
		sort.SliceStable(s.Postcodes, func(i, j int) bool {
			return s.Postcodes[i].Postcode < s.Postcodes[j].Postcode || (s.Postcodes[i].Postcode == s.Postcodes[j].Postcode && s.Postcodes[i].City < s.Postcodes[j].City)
		})
		toExport = append(toExport, s)
	}
	sort.SliceStable(toExport, func(i, j int) bool { return toExport[i].State < toExport[j].State })
	return e.enc.Encode(toExport)
}
