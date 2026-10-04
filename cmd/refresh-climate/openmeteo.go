package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/mpangarova/mpangarova.github.io/climate/data"
)

const (
	archiveAPI = "https://archive-api.open-meteo.com/v1/archive"
	climateAPI = "https://climate-api.open-meteo.com/v1/climate"
	model      = "EC_Earth3P_HR"

	maxAttempts = 4
	retryWait   = 2 * time.Minute
)

var client = &http.Client{Timeout: 3 * time.Minute}

// apiResponse is the part of an Open-Meteo reply we read.
// Missing days are null, hence the pointers.
type apiResponse struct {
	Daily struct {
		Time  []string   `json:"time"`
		Mean  []*float64 `json:"temperature_2m_mean"`
		Max   []*float64 `json:"temperature_2m_max"`
		Min   []*float64 `json:"temperature_2m_min"`
		Model []*float64 `json:"temperature_2m_mean_EC_Earth3P_HR"`
	} `json:"daily"`
}

// fetchObservations gets daily mean, max and min temperatures for first..last.
func fetchObservations(place *data.Place, first, last int) (*apiResponse, error) {
	return fetch(archiveAPI, url.Values{
		"latitude":   {fmt.Sprint(place.Lat)},
		"longitude":  {fmt.Sprint(place.Lon)},
		"start_date": {fmt.Sprintf("%d-01-01", first)},
		"end_date":   {fmt.Sprintf("%d-12-31", last)},
		"daily":      {"temperature_2m_mean,temperature_2m_max,temperature_2m_min"},
		"timezone":   {"Europe/Sofia"},
	})
}

// fetchProjection gets the model's daily mean temperature, 1950–2050.
func fetchProjection(place *data.Place) (*apiResponse, error) {
	return fetch(climateAPI, url.Values{
		"latitude":   {fmt.Sprint(place.Lat)},
		"longitude":  {fmt.Sprint(place.Lon)},
		"start_date": {"1950-01-01"},
		"end_date":   {"2050-12-31"},
		"models":     {model},
		"daily":      {"temperature_2m_mean"},
	})
}

// fetch calls Open-Meteo, retrying on rate limits and server errors.
func fetch(endpoint string, query url.Values) (*apiResponse, error) {
	requestURL := endpoint + "?" + query.Encode()
	for attempt := 1; ; attempt++ {
		resp, err := client.Get(requestURL)
		if err != nil {
			return nil, err
		}
		body, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close() // already read; a close error changes nothing
		if err != nil {
			return nil, err
		}
		if resp.StatusCode == http.StatusOK {
			var response apiResponse
			if err := json.Unmarshal(body, &response); err != nil {
				return nil, err
			}
			return &response, nil
		}
		retryable := resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500
		if retryable && attempt < maxAttempts {
			log.Printf("HTTP %d, retrying in %s", resp.StatusCode, retryWait)
			time.Sleep(retryWait)
			continue
		}
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, body)
	}
}
