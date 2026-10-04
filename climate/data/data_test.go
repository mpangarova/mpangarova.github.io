package data

import "testing"

// The tests the page prints as `go test ./climate/data -v`, run on the
// data.json next to them: the exact file GitHub Pages serves.

func load(t *testing.T) *Set {
	t.Helper()
	set, err := Load("data.json")
	if err != nil {
		t.Fatal(err)
	}
	return set
}

func TestEveryYearIsThere(t *testing.T) {
	if err := EveryYearIsThere(load(t)); err != nil {
		t.Error(err)
	}
}

func TestNoMissingDays(t *testing.T) {
	if err := NoMissingDays(load(t)); err != nil {
		t.Error(err)
	}
}

func TestTemperaturesArePlausible(t *testing.T) {
	if err := TemperaturesArePlausible(load(t)); err != nil {
		t.Error(err)
	}
}

func TestDayCountsFitInAYear(t *testing.T) {
	if err := DayCountsFitInAYear(load(t)); err != nil {
		t.Error(err)
	}
}

func TestModelMatchesObservations(t *testing.T) {
	if err := ModelMatchesObservations(load(t)); err != nil {
		t.Error(err)
	}
}

// The checks must catch broken data, or they prove nothing.
func TestChecksCatchBrokenData(t *testing.T) {
	broken := func(edit func(*Set)) *Set {
		set := load(t)
		edit(set)
		return set
	}
	first := func(set *Set) *Place { return set.Places[set.Order[0]] }
	cases := []struct {
		name  string
		check func(*Set) error
		edit  func(*Set)
	}{
		{"missing year", EveryYearIsThere, func(set *Set) { place := first(set); place.Years = place.Years[1:] }},
		{"short year", NoMissingDays, func(set *Set) { first(set).Days[10] = 300 }},
		{"impossible mean", TemperaturesArePlausible, func(set *Set) { first(set).Temp[5] = 40 }},
		{"negative count", DayCountsFitInAYear, func(set *Set) { first(set).Hot[3] = -1 }},
		{"model drift", ModelMatchesObservations, func(set *Set) {
			place := first(set)
			for i := range place.Proj.Temp {
				place.Proj.Temp[i] += 0.5
			}
		}},
	}
	for _, tc := range cases {
		if err := tc.check(broken(tc.edit)); err == nil {
			t.Errorf("%s: check passed on broken data", tc.name)
		}
	}
}
