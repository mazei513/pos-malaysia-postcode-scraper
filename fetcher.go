package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type fetcher struct {
	cache cacheManager
	log   logger

	interval    time.Duration
	ignoreCache bool
}

func (f fetcher) get(postcode string) ([]apiResponse, error) {
	if !f.ignoreCache {
		ars, err := f.cache.get(postcode)
		if err == nil {
			f.log.println("Found cached response for", postcode)
			return ars, nil
		}
	}

	f.log.println("Fetching for", postcode)
	u := fmt.Sprintf("https://api.pos.com.my/PostcodeWebApi/api/Postcode?Postcode=%s", postcode)
	resp, err := http.Get(u)
	if err != nil {
		return nil, fmt.Errorf("fetch postcode: %w", err)
	}

	time.Sleep(f.interval)

	ars := []apiResponse{}
	err = json.NewDecoder(resp.Body).Decode(&ars)
	if err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	// ignore caching errors
	f.cache.set(postcode, ars)

	return ars, nil
}
