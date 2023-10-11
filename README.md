# Concurrent web scraper with primitive cache

## How to run
*A prerequisite is to have Go 1.21+ installed*
In root repo dir, run:
```bash
go run .
```

Sites sourced from `urls` will get scraped, with the retrieved words displayed on
`localhost:1337/metrics`

## Issues
As of writing this README, the scraper runs an unholy amount of Goroutines.