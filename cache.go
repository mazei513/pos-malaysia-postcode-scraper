package main

import (
	"encoding/json"
	"os"
	"path"
)

type cacheManager struct {
	path string
}

func (c cacheManager) makeFolder() error {
	if c.path == "" {
		return nil
	}
	return os.MkdirAll(c.path, 0777)
}

func (c cacheManager) get(postcode string) ([]apiResponse, error) {
	if c.path == "" {
		return []apiResponse{}, nil
	}

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
	if c.path == "" {
		return nil
	}

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
