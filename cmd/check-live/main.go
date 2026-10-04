// Command check-live checks that the published site serves what the repo
// holds: both pages load, and the live data.json passes the checks and
// matches the repo's file byte for byte. It retries while Pages publishes.
//
//	go run ./cmd/check-live            # try for up to 10 minutes
//	go run ./cmd/check-live -wait 0    # one try
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/mpangarova/mpangarova.github.io/climate/data"
)

var pages = []string{"/", "/climate/"}

const dataPath = "/climate/data/data.json"

var client = &http.Client{Timeout: 30 * time.Second}

func main() {
	site := flag.String("site", "https://mpangarova.github.io", "published site to check")
	path := flag.String("data", "climate/data/data.json", "the dataset the site should serve")
	wait := flag.Duration("wait", 10*time.Minute, "how long to keep trying while Pages publishes")
	every := flag.Duration("every", 30*time.Second, "pause between tries")
	flag.Parse()

	want, err := os.ReadFile(*path)
	if err != nil {
		log.Fatal(err)
	}
	deadline := time.Now().Add(*wait)
	for {
		err := check(*site, want)
		if err == nil {
			log.Printf("%s serves both pages and the same data.json as the repository", *site)
			return
		}
		if time.Now().After(deadline) {
			log.Fatal(err)
		}
		log.Printf("%v; trying again in %s", err, *every)
		time.Sleep(*every)
	}
}

// check loads the pages and the live data.json, validates it and compares it with want.
func check(site string, want []byte) error {
	for _, page := range pages {
		if _, err := get(site + page); err != nil {
			return err
		}
	}
	live, err := get(site + dataPath)
	if err != nil {
		return err
	}
	var set data.Set
	if err := json.Unmarshal(live, &set); err != nil {
		return fmt.Errorf("live data.json: %w", err)
	}
	if err := data.Validate(&set); err != nil {
		return fmt.Errorf("live data.json failed its checks:\n%w", err)
	}
	if !bytes.Equal(live, want) {
		return fmt.Errorf("live data.json (retrieved %s) is not the one in the repository yet", set.Meta.Retrieved)
	}
	return nil
}

// get fetches a URL and fails on anything but 200.
// The query string bypasses the CDN cache.
func get(url string) ([]byte, error) {
	resp, err := client.Get(url + "?check=" + strconv.FormatInt(time.Now().UnixNano(), 10))
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: HTTP %d", url, resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}
