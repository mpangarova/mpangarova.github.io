// Command refresh-climate turns Open-Meteo's daily measurements into yearly
// data and writes the file only if every check passes. Otherwise it exits
// with an error and the published data stays as it was.
//
//	go run ./cmd/refresh-climate              # refresh the last two years and any new ones
//	go run ./cmd/refresh-climate -full        # fetch every year again
//	go run ./cmd/refresh-climate -until 2025  # stop at a given year
//
// Files:
//
//	main.go       flags and the refresh loop
//	openmeteo.go  HTTP calls to Open-Meteo, with retries
//	aggregate.go  daily values → yearly counts, and model alignment
//	merge.go      which years to fetch, and joining them with the file
package main

import (
	"flag"
	"log"
	"time"

	"github.com/mpangarova/mpangarova.github.io/climate/data"
)

func main() {
	path := flag.String("data", "climate/data/data.json", "dataset to refresh")
	until := flag.Int("until", time.Now().UTC().Year()-1, "last complete year to include")
	pause := flag.Duration("pause", 10*time.Second, "wait between places (Open-Meteo rate limit)")
	refetch := flag.Int("refetch", 2, "recent years to fetch again; older years are kept from the file")
	full := flag.Bool("full", false, "fetch every year again, not just the recent ones")
	flag.Parse()

	old, err := data.Load(*path)
	if err != nil {
		log.Fatal(err)
	}

	fresh := &data.Set{
		Meta: data.Meta{
			FirstYear: old.Meta.FirstYear,
			LastYear:  *until,
			Retrieved: time.Now().UTC().Format("2006-01-02"),
			Source:    old.Meta.Source,
		},
		Order:  old.Order,
		Places: map[string]*data.Place{},
	}

	for i, key := range old.Order {
		if i > 0 {
			time.Sleep(*pause)
		}
		prev := old.Places[key]
		place, err := refreshPlace(prev, old.Meta.LastYear, fresh.Meta.FirstYear, fresh.Meta.LastYear, *refetch, *full, *pause)
		if err != nil {
			log.Fatalf("%s: %v", prev.Name, err)
		}
		fresh.Places[key] = place
	}

	if err := data.Validate(fresh); err != nil {
		log.Fatalf("new data failed its checks, keeping the published file:\n%v", err)
	}
	if err := fresh.Save(*path); err != nil {
		log.Fatal(err)
	}
	log.Printf("all checks passed, wrote %s", *path)
}

// refreshPlace fetches recent years for one place, merges them with the
// file, and fetches the model run if the place has none.
func refreshPlace(prev *data.Place, oldLast, first, last, refetch int, full bool, pause time.Duration) (*data.Place, error) {
	from := fetchFrom(prev, oldLast, first, last, refetch, full)
	log.Printf("%s: fetching %d–%d", prev.Name, from, last)
	response, err := fetchObservations(prev, from, last)
	if err != nil {
		return nil, err
	}
	recent := aggregate(prev, response, from, last)
	place, err := merge(prev, recent, first, last)
	if err != nil {
		return nil, err
	}
	// The model run never changes: fetch it only for new places.
	place.Proj = prev.Proj
	if place.Proj == nil {
		time.Sleep(pause)
		modelResponse, err := fetchProjection(place)
		if err != nil {
			return nil, err
		}
		if place.Proj, err = alignProjection(place, modelResponse); err != nil {
			return nil, err
		}
	}
	return place, nil
}
