# mpangarova.github.io

Personal site of Maria Pangarova, served by GitHub Pages at https://mpangarova.github.io.

## What's here

```
index.html                    Homepage
img/cliffs.jpg                Header photo (film)
climate/
  index.html                  Climate Lab: "How much warmer is Bulgaria?"
  data/
    data.json                 Yearly data the page loads
    data.go                   Thresholds, limits and the checks the data has to pass
    data_test.go              The same checks as Go tests
    page_test.go              Keeps the page's JavaScript checks in line with the Go ones
cmd/refresh-climate/
  main.go                     Flags and the refresh loop
  openmeteo.go                Calls to Open-Meteo, with retries
  aggregate.go                Daily values to yearly counts; model alignment
  merge.go                    Which years to fetch; joining them with the file
  *_test.go                   Tests for all of the above, no network needed
.github/workflows/
  climate-data.yml            Runs the refresh once a month
go.mod
_config.yml                   Keeps the Go files out of the published site
```

Both pages are plain HTML with inline CSS and JavaScript. There is no build step.

## Climate Lab data

The page shows yearly aggregates of daily ERA5 reanalysis data from the
[Open-Meteo Historical Weather API](https://open-meteo.com/en/docs/historical-weather-api),
and a climate model run (EC-Earth3P-HR) from the
[Open-Meteo Climate API](https://open-meteo.com/en/docs/climate-api).

On the 3rd of every month, GitHub Actions:

1. runs `go test ./... -v` on the code and the data that is live,
2. runs `go run ./cmd/refresh-climate` to pull the last two years again (plus any new year) and check the whole set,
3. runs the tests again on the new file,
4. commits `climate/data/data.json` only if everything passed.

If any check fails, the run fails and the site keeps the last good data.

Run it locally:

```sh
go test ./... -v
go run ./cmd/refresh-climate         # last two years and any new ones
go run ./cmd/refresh-climate -full   # every year again (heavy, use rarely)
```

Older years are kept from `data.json`, because past reanalysis data does not change.
A full refresh downloads 75+ years per place and can hit Open-Meteo's rate limits.
