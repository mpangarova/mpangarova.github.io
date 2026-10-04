package main

import (
	"fmt"
	"math"

	"github.com/mpangarova/mpangarova.github.io/climate/data"
)

// minModelDays is how many daily values a model year needs to count as complete.
const minModelDays = 360

func round2(value float64) float64 { return math.Round(value*100) / 100 }

// yearOf reads the year from an ISO date such as "2025-07-14".
func yearOf(day string) (int, bool) {
	var year int
	if _, err := fmt.Sscanf(day, "%d-", &year); err != nil {
		return 0, false
	}
	return year, true
}

// aggregate turns daily measurements into yearly values for first..last.
// Days with a missing value are skipped, which leaves that year short;
// data.NoMissingDays then refuses to publish it.
func aggregate(base *data.Place, response *apiResponse, first, last int) *data.Place {
	yearCount := last - first + 1
	place := &data.Place{Name: base.Name, Lat: base.Lat, Lon: base.Lon,
		Years: make([]int, yearCount), Temp: make([]float64, yearCount), Hot: make([]int, yearCount),
		Trop: make([]int, yearCount), Frost: make([]int, yearCount), Days: make([]int, yearCount)}
	sums := make([]float64, yearCount)
	for i := range place.Years {
		place.Years[i] = first + i
	}
	daily := response.Daily
	for i, day := range daily.Time {
		year, ok := yearOf(day)
		if !ok || year < first || year > last || i >= len(daily.Mean) || i >= len(daily.Max) || i >= len(daily.Min) {
			continue
		}
		offset := year - first
		mean, high, low := daily.Mean[i], daily.Max[i], daily.Min[i]
		if mean == nil || high == nil || low == nil {
			continue
		}
		sums[offset] += *mean
		place.Days[offset]++
		if *high >= data.HotDayThreshold {
			place.Hot[offset]++
		}
		if *low >= data.TropicalNightThreshold {
			place.Trop[offset]++
		}
		if *low < data.FrostThreshold {
			place.Frost[offset]++
		}
	}
	for offset := range sums {
		if place.Days[offset] > 0 {
			place.Temp[offset] = round2(sums[offset] / float64(place.Days[offset]))
		}
	}
	return place
}

// alignProjection turns the model's daily values into yearly means and aligns
// them to the observations with the delta method: the model's anomaly against
// its own baseline mean is added to the observed baseline mean.
func alignProjection(place *data.Place, response *apiResponse) (*data.Projection, error) {
	values := response.Daily.Mean
	if values == nil {
		values = response.Daily.Model
	}
	sums, counts := map[int]float64{}, map[int]int{}
	for i, day := range response.Daily.Time {
		year, ok := yearOf(day)
		if !ok || i >= len(values) || values[i] == nil {
			continue
		}
		sums[year] += *values[i]
		counts[year]++
	}
	var years []int
	for year := 1950; year <= 2050; year++ {
		if counts[year] >= minModelDays {
			years = append(years, year)
		}
	}
	baselineYears := data.BaselineTo - data.BaselineFrom + 1
	var modelBase, observedBase float64
	var modelYears, observedYears int
	for _, year := range years {
		if year >= data.BaselineFrom && year <= data.BaselineTo {
			modelBase += sums[year] / float64(counts[year])
			modelYears++
		}
	}
	for i, year := range place.Years {
		if year >= data.BaselineFrom && year <= data.BaselineTo {
			observedBase += place.Temp[i]
			observedYears++
		}
	}
	if modelYears != baselineYears || observedYears != baselineYears {
		return nil, fmt.Errorf("%d–%d is incomplete (model %d, observed %d years)",
			data.BaselineFrom, data.BaselineTo, modelYears, observedYears)
	}
	modelBase /= float64(baselineYears)
	observedBase /= float64(baselineYears)
	aligned := &data.Projection{}
	for _, year := range years {
		aligned.Years = append(aligned.Years, year)
		aligned.Temp = append(aligned.Temp, round2(observedBase+sums[year]/float64(counts[year])-modelBase))
	}
	return aligned, nil
}
