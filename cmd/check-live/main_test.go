package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// site fakes GitHub Pages: the two pages and the given data.json.
func site(t *testing.T, liveData []byte, missing string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == missing {
			http.NotFound(w, r)
			return
		}
		switch r.URL.Path {
		case dataPath:
			_, _ = w.Write(liveData)
		case "/", "/climate/":
			_, _ = w.Write([]byte("<!DOCTYPE html>"))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

func repoData(t *testing.T) []byte {
	t.Helper()
	contents, err := os.ReadFile("../../climate/data/data.json")
	if err != nil {
		t.Fatal(err)
	}
	return contents
}

func TestCheckPassesWhenLiveMatchesRepository(t *testing.T) {
	want := repoData(t)
	if err := check(site(t, want, "").URL, want); err != nil {
		t.Error(err)
	}
}

func TestCheckCatchesDataNotPublishedYet(t *testing.T) {
	want := repoData(t)
	older := bytes.Replace(want, []byte(`"retrieved":"`), []byte(`"retrieved":"x`), 1)
	err := check(site(t, older, "").URL, want)
	if err == nil || !strings.Contains(err.Error(), "not the one in the repository") {
		t.Errorf("got %v, want a not-published-yet error", err)
	}
}

func TestCheckCatchesBrokenLiveData(t *testing.T) {
	want := repoData(t)
	broken := bytes.Replace(want, []byte(`"firstYear":1950`), []byte(`"firstYear":1949`), 1)
	err := check(site(t, broken, "").URL, want)
	if err == nil || !strings.Contains(err.Error(), "failed its checks") {
		t.Errorf("got %v, want a failed-checks error", err)
	}
}

func TestCheckCatchesMissingPage(t *testing.T) {
	want := repoData(t)
	err := check(site(t, want, "/climate/").URL, want)
	if err == nil || !strings.Contains(err.Error(), "HTTP 404") {
		t.Errorf("got %v, want a 404 error", err)
	}
}
