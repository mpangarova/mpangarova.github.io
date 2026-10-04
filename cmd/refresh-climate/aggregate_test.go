package main

import (
	"fmt"
	"math"
	"testing"

	"github.com/mpangarova/mpangarova.github.io/climate/data"
)

func ptr(value float64) *float64 { return &value }

// day is one row of a fake Open-Meteo reply; a nil mean marks a missing day.
type day struct {
	date           string
	mean, max, min *float64
}

func response(days ...day) *apiResponse {
	var r apiResponse
	for _, d := range days {
		r.Daily.Time = append(r.Daily.Time, d.date)
		r.Daily.Mean = append(r.Daily.Mean, d.mean)
		r.Daily.Max = append(r.Daily.Max, d.max)
		r.Daily.Min = append(r.Daily.Min, d.min)
	}
	return &r
}

func TestAggregateCountsEachKindOfDay(t *testing.T) {
	base := &data.Place{Name: "Test"}
	got := aggregate(base, response(
		day{"2024-07-01", ptr(25), ptr(31), ptr(21)},   // hot day and tropical night
		day{"2024-07-02", ptr(24), ptr(30), ptr(20)},   // at the thresholds: still hot and tropical
		day{"2024-07-03", ptr(20), ptr(29.9), ptr(19)}, // neither
		day{"2024-01-10", ptr(-2), ptr(1), ptr(-0.1)},  // frost
		day{"2024-01-11", ptr(1), ptr(4), ptr(0)},      // 0 °C is not frost
		day{"2024-01-12", nil, ptr(5), ptr(1)},         // missing value: skipped
		day{"2023-12-31", ptr(9), ptr(35), ptr(25)},    // outside the range: ignored
		day{"2025-03-01", ptr(10), ptr(15), ptr(5)},
	), 2024, 2025)

	if len(got.Years) != 2 || got.Years[0] != 2024 || got.Years[1] != 2025 {
		t.Fatalf("years = %v", got.Years)
	}
	checks := []struct {
		name      string
		got, want int
	}{
		{"2024 days", got.Days[0], 5},
		{"2024 hot", got.Hot[0], 2},
		{"2024 tropical", got.Trop[0], 2},
		{"2024 frost", got.Frost[0], 1},
		{"2025 days", got.Days[1], 1},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s = %d, want %d", c.name, c.got, c.want)
		}
	}
	if want := round2((25 + 24 + 20 - 2 + 1) / 5.0); got.Temp[0] != want {
		t.Errorf("2024 mean = %v, want %v", got.Temp[0], want)
	}
	if got.Temp[1] != 10 {
		t.Errorf("2025 mean = %v, want 10", got.Temp[1])
	}
}

// modelReply fakes a climate API reply: valueFor(year) on every day, from..to.
func modelReply(from, to int, valueFor func(year int) float64) *apiResponse {
	var r apiResponse
	for year := from; year <= to; year++ {
		for d := 1; d <= 365; d++ {
			r.Daily.Time = append(r.Daily.Time, fmt.Sprintf("%d-%03d", year, d))
			r.Daily.Mean = append(r.Daily.Mean, ptr(valueFor(year)))
		}
	}
	return &r
}

func observedPlace(temp float64) *data.Place {
	place := &data.Place{Name: "Test"}
	for year := 1950; year <= 2025; year++ {
		place.Years = append(place.Years, year)
		place.Temp = append(place.Temp, temp)
	}
	return place
}

func TestAlignProjectionShiftsModelOntoObservations(t *testing.T) {
	// The model runs 2 °C cold on the baseline. Aligned, it should sit on the
	// observed 12 °C and keep its own warming trend.
	reply := modelReply(1950, 2050, func(year int) float64 {
		if year > 2020 {
			return 10 + float64(year-2020)*0.1
		}
		return 10
	})
	got, err := alignProjection(observedPlace(12), reply)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Years) != 101 {
		t.Fatalf("%d years, want 101", len(got.Years))
	}
	at := func(year int) float64 { return got.Temp[year-1950] }
	if at(2000) != 12 {
		t.Errorf("2000 = %v, want 12", at(2000))
	}
	if math.Abs(at(2050)-15) > 1e-9 {
		t.Errorf("2050 = %v, want 15", at(2050))
	}
}

func TestAlignProjectionSkipsIncompleteYears(t *testing.T) {
	reply := modelReply(1950, 2050, func(int) float64 { return 10 })
	// Drop most of 2050: it should not appear in the projection.
	cut := len(reply.Daily.Time) - 300
	reply.Daily.Time, reply.Daily.Mean = reply.Daily.Time[:cut], reply.Daily.Mean[:cut]
	got, err := alignProjection(observedPlace(12), reply)
	if err != nil {
		t.Fatal(err)
	}
	if last := got.Years[len(got.Years)-1]; last != 2049 {
		t.Errorf("last year = %d, want 2049", last)
	}
}

func TestAlignProjectionRefusesMissingBaseline(t *testing.T) {
	reply := modelReply(2000, 2050, func(int) float64 { return 10 }) // no 1991–1999
	if _, err := alignProjection(observedPlace(12), reply); err == nil {
		t.Error("aligned a model with an incomplete baseline")
	}
}
