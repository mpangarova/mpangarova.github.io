package data

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

// The page runs its own JavaScript copy of the checks, so visitors see them
// pass without trusting this repo. This test fails if a rule, limit or
// threshold label on the page drifts from the constants here.
func TestPageRunsTheSameChecks(t *testing.T) {
	contents, err := os.ReadFile("../index.html")
	if err != nil {
		t.Fatal(err)
	}
	page := string(contents)

	expectations := map[string]string{}
	for _, check := range Checks {
		expectations["runs "+check.Name] = fmt.Sprintf("test('%s'", check.Name)
	}
	for name, snippet := range map[string]string{
		"plausible mean range": fmt.Sprintf("t>%g&&t<%g", PlausibleMeanLow, PlausibleMeanHigh),
		"day count range":      fmt.Sprintf("x>=0&&x<=%d", MaxDaysInYear),
		"alignment baseline":   fmt.Sprintf("y>=%d&&y<=%d", BaselineFrom, BaselineTo),
		"alignment tolerance":  fmt.Sprintf("Math.abs(a-b)<%g", AlignmentTolerance),
		"first published year": fmt.Sprintf("LAST>=%d", FirstPublishedYear),
		"leap years":           "(y%4===0&&y%100!==0)||y%400===0",
		"hot day label":        fmt.Sprintf("≥%g°C", HotDayThreshold),
		"tropical night label": fmt.Sprintf("≥%g°C", TropicalNightThreshold),
		"frost day label":      fmt.Sprintf("&lt;%g°C", FrostThreshold),
	} {
		expectations[name] = snippet
	}

	for name, snippet := range expectations {
		if !strings.Contains(page, snippet) {
			t.Errorf("%s: climate/index.html does not contain %q", name, snippet)
		}
	}
}
