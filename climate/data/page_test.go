package data

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

// The page runs its own copy of the checks in JavaScript, so a visitor sees
// them pass without trusting this repository. That copy must not drift from
// the Go rules. This test reads the page and fails if a rule, a limit or a
// threshold label no longer matches the constants in this package.
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
