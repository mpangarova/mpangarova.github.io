package main

import (
	"testing"

	"github.com/mpangarova/mpangarova.github.io/climate/data"
)

func place(first, last int, temp float64) *data.Place {
	p := &data.Place{Name: "Test"}
	for y := first; y <= last; y++ {
		p.Years = append(p.Years, y)
		p.Temp = append(p.Temp, temp)
		p.Hot = append(p.Hot, 1)
		p.Trop = append(p.Trop, 2)
		p.Frost = append(p.Frost, 3)
		p.Days = append(p.Days, 365)
	}
	return p
}

func TestFetchFrom(t *testing.T) {
	prev := place(1950, 2025, 12)
	cases := []struct {
		name               string
		prev               *data.Place
		oldLast, last, ref int
		full               bool
		want               int
	}{
		{"same year, refetch two", prev, 2025, 2025, 2, false, 2024},
		{"one new year", prev, 2025, 2026, 2, false, 2025},
		{"skipped a year", prev, 2025, 2028, 2, false, 2026},
		{"full refresh", prev, 2025, 2026, 2, true, 1950},
		{"no history", &data.Place{}, 2025, 2026, 2, false, 1950},
	}
	for _, c := range cases {
		if got := fetchFrom(c.prev, c.oldLast, 1950, c.last, c.ref, c.full); got != c.want {
			t.Errorf("%s: fetchFrom = %d, want %d", c.name, got, c.want)
		}
	}
}

func TestMergeKeepsOldYearsAndTakesNewOnes(t *testing.T) {
	prev := place(1950, 2025, 12)
	recent := place(2025, 2026, 14)
	got, err := merge(prev, recent, 1950, 2026)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Years) != 77 || got.Years[0] != 1950 || got.Years[76] != 2026 {
		t.Fatalf("years %d..%d (%d)", got.Years[0], got.Years[len(got.Years)-1], len(got.Years))
	}
	if got.Temp[0] != 12 || got.Temp[74] != 12 {
		t.Errorf("old years should come from the file, got %v and %v", got.Temp[0], got.Temp[74])
	}
	if got.Temp[75] != 14 || got.Temp[76] != 14 {
		t.Errorf("recent years should come from the fetch, got %v and %v", got.Temp[75], got.Temp[76])
	}
}

func TestMergeRefusesGaps(t *testing.T) {
	prev := place(1950, 2023, 12)
	recent := place(2025, 2026, 14) // 2024 is missing from both
	if _, err := merge(prev, recent, 1950, 2026); err == nil {
		t.Error("merge accepted a missing year")
	}
}
