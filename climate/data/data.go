// Package data holds the Climate Lab dataset and the checks it has to pass
// before it is published. The same checks run in the browser on every page
// load, and as Go tests in CI before any new data is committed.
package data

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
)

// Thresholds for the yearly day counts, in °C.
const (
	HotDayThreshold        = 30.0 // a day is hot when its maximum reaches this
	TropicalNightThreshold = 20.0 // a night is tropical when its minimum stays at or above this
	FrostThreshold         = 0.0  // a day has frost when its minimum drops below this
)

// Limits the checks enforce.
const (
	PlausibleMeanLow   = 2.0  // lowest believable annual mean, °C
	PlausibleMeanHigh  = 18.0 // highest believable annual mean, °C
	MaxDaysInYear      = 366
	BaselineFrom       = 1991 // reference period the model is aligned on
	BaselineTo         = 2020
	AlignmentTolerance = 0.05 // allowed model–observation gap on the baseline, °C
	FirstPublishedYear = 2025 // the data must never end before this year
)

// Projection is a climate model run, aligned to the observations.
type Projection struct {
	Years []int     `json:"years"`
	Temp  []float64 `json:"temp"`
}

// Place is one location with yearly aggregates of daily measurements.
type Place struct {
	Name  string      `json:"name"`
	Lat   float64     `json:"lat"`
	Lon   float64     `json:"lon"`
	Years []int       `json:"years"`
	Temp  []float64   `json:"temp"`  // annual mean temperature, °C
	Hot   []int       `json:"hot"`   // days at or above HotDayThreshold
	Trop  []int       `json:"trop"`  // nights at or above TropicalNightThreshold
	Frost []int       `json:"frost"` // days below FrostThreshold
	Days  []int       `json:"days"`  // days with a measurement
	Proj  *Projection `json:"proj,omitempty"`
}

// Meta describes the dataset as a whole.
type Meta struct {
	FirstYear int    `json:"firstYear"`
	LastYear  int    `json:"lastYear"`
	Retrieved string `json:"retrieved"`
	Source    string `json:"source"`
}

// Set is the whole file the page reads.
type Set struct {
	Meta   Meta              `json:"meta"`
	Order  []string          `json:"order"`
	Places map[string]*Place `json:"places"`
}

// Load reads a dataset from disk.
func Load(path string) (*Set, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var set Set
	if err := json.Unmarshal(contents, &set); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return &set, nil
}

// Save writes the dataset atomically, so a failed write never leaves half a file behind.
func (s *Set) Save(path string) error {
	encoded, err := json.Marshal(s)
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(encoded, '\n'), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// ordered returns the places in the order the page shows them.
func (s *Set) ordered() ([]*Place, error) {
	if len(s.Order) == 0 {
		return nil, errors.New("no places")
	}
	places := make([]*Place, 0, len(s.Order))
	for _, key := range s.Order {
		place, ok := s.Places[key]
		if !ok {
			return nil, fmt.Errorf("place %q is listed but missing", key)
		}
		places = append(places, place)
	}
	return places, nil
}

// Check is one rule the data has to follow.
type Check struct {
	Name string
	Run  func(*Set) error
}

// Checks are the same five tests the page shows in its terminal.
var Checks = []Check{
	{"TestEveryYearIsThere", EveryYearIsThere},
	{"TestNoMissingDays", NoMissingDays},
	{"TestTemperaturesArePlausible", TemperaturesArePlausible},
	{"TestDayCountsFitInAYear", DayCountsFitInAYear},
	{"TestModelMatchesObservations", ModelMatchesObservations},
}

// Validate runs every check and reports all failures together.
func Validate(set *Set) error {
	var errs []error
	for _, check := range Checks {
		if err := check.Run(set); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", check.Name, err))
		}
	}
	return errors.Join(errs...)
}

// EveryYearIsThere: every place covers firstYear..lastYear with no gaps, and every series has one value per year.
func EveryYearIsThere(set *Set) error {
	places, err := set.ordered()
	if err != nil {
		return err
	}
	if set.Meta.LastYear < FirstPublishedYear {
		return fmt.Errorf("last year %d is older than the first published data", set.Meta.LastYear)
	}
	want := set.Meta.LastYear - set.Meta.FirstYear + 1
	for _, place := range places {
		if len(place.Years) != want {
			return fmt.Errorf("%s: %d years, want %d", place.Name, len(place.Years), want)
		}
		for i, year := range place.Years {
			if year != set.Meta.FirstYear+i {
				return fmt.Errorf("%s: year %d at position %d", place.Name, year, i)
			}
		}
		for name, count := range map[string]int{"temp": len(place.Temp), "hot": len(place.Hot), "trop": len(place.Trop), "frost": len(place.Frost), "days": len(place.Days)} {
			if count != want {
				return fmt.Errorf("%s: %s has %d values, want %d", place.Name, name, count, want)
			}
		}
	}
	return nil
}

func isLeap(year int) bool { return year%4 == 0 && (year%100 != 0 || year%400 == 0) }

// NoMissingDays: every year is built from 365 or 366 daily measurements.
func NoMissingDays(set *Set) error {
	places, err := set.ordered()
	if err != nil {
		return err
	}
	for _, place := range places {
		for i, days := range place.Days {
			want := 365
			if isLeap(place.Years[i]) {
				want = 366
			}
			if days != want {
				return fmt.Errorf("%s %d: %d days, want %d", place.Name, place.Years[i], days, want)
			}
		}
	}
	return nil
}

// TemperaturesArePlausible: annual means stay between PlausibleMeanLow and PlausibleMeanHigh.
func TemperaturesArePlausible(set *Set) error {
	places, err := set.ordered()
	if err != nil {
		return err
	}
	for _, place := range places {
		for i, temp := range place.Temp {
			if !(temp > PlausibleMeanLow && temp < PlausibleMeanHigh) {
				return fmt.Errorf("%s %d: mean %.2f °C", place.Name, place.Years[i], temp)
			}
		}
	}
	return nil
}

// DayCountsFitInAYear: hot, tropical and frost days are all within 0–MaxDaysInYear.
func DayCountsFitInAYear(set *Set) error {
	places, err := set.ordered()
	if err != nil {
		return err
	}
	for _, place := range places {
		for name, series := range map[string][]int{"hot": place.Hot, "trop": place.Trop, "frost": place.Frost} {
			for i, count := range series {
				if count < 0 || count > MaxDaysInYear {
					return fmt.Errorf("%s %d: %d %s days", place.Name, place.Years[i], count, name)
				}
			}
		}
	}
	return nil
}

func meanBetween(years []int, values []float64, from, to int) (float64, bool) {
	sum, count := 0.0, 0
	for i, year := range years {
		if year >= from && year <= to {
			sum += values[i]
			count++
		}
	}
	return sum / float64(count), count == to-from+1
}

// ModelMatchesObservations: the model is aligned on the baseline within AlignmentTolerance.
func ModelMatchesObservations(set *Set) error {
	places, err := set.ordered()
	if err != nil {
		return err
	}
	for _, place := range places {
		if place.Proj == nil {
			continue
		}
		observed, observedComplete := meanBetween(place.Years, place.Temp, BaselineFrom, BaselineTo)
		modelled, modelComplete := meanBetween(place.Proj.Years, place.Proj.Temp, BaselineFrom, BaselineTo)
		if !observedComplete || !modelComplete {
			return fmt.Errorf("%s: %d–%d is incomplete", place.Name, BaselineFrom, BaselineTo)
		}
		if diff := math.Abs(observed - modelled); diff >= AlignmentTolerance {
			return fmt.Errorf("%s: model is %.3f °C off on %d–%d", place.Name, diff, BaselineFrom, BaselineTo)
		}
	}
	return nil
}
