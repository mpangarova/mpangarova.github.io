package main

import (
	"fmt"

	"github.com/mpangarova/mpangarova.github.io/climate/data"
)

// fetchFrom picks the first year to download. Past reanalysis data never
// changes, so only recent and new years are fetched again, unless the place
// has no usable history.
func fetchFrom(prev *data.Place, oldLast, first, last, refetch int, full bool) int {
	if full || len(prev.Years) == 0 || prev.Years[0] != first {
		return first
	}
	from := last - refetch + 1
	if oldLast+1 < from {
		from = oldLast + 1
	}
	if from < first {
		from = first
	}
	return from
}

// merge takes years before recent.Years[0] from prev and the rest from
// recent, covering first..last with no gaps.
func merge(prev, recent *data.Place, first, last int) (*data.Place, error) {
	yearCount := last - first + 1
	out := &data.Place{Name: prev.Name, Lat: prev.Lat, Lon: prev.Lon,
		Years: make([]int, yearCount), Temp: make([]float64, yearCount), Hot: make([]int, yearCount),
		Trop: make([]int, yearCount), Frost: make([]int, yearCount), Days: make([]int, yearCount)}
	for offset := 0; offset < yearCount; offset++ {
		year := first + offset
		source, index := recent, year-firstYear(recent)
		if len(recent.Years) == 0 || year < recent.Years[0] {
			source, index = prev, year-firstYear(prev)
		}
		if index < 0 || index >= len(source.Years) || source.Years[index] != year {
			return nil, fmt.Errorf("no data for %d", year)
		}
		out.Years[offset], out.Temp[offset], out.Hot[offset] = year, source.Temp[index], source.Hot[index]
		out.Trop[offset], out.Frost[offset], out.Days[offset] = source.Trop[index], source.Frost[index], source.Days[index]
	}
	return out, nil
}

func firstYear(place *data.Place) int {
	if len(place.Years) == 0 {
		return 0
	}
	return place.Years[0]
}
