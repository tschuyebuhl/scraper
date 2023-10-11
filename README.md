# Concurrent web scraper with primitive cache

## How to run
*A prerequisite is to have Go 1.21+ installed*
In root repo dir, run:
```bash
go run .
```
Or, using docker, run:
```bash
docker build -t scraper .
docker run -d -p 1337:1337 scraper
```
Above commands will build and run the container, binding your host port 1337 to the same port on the container.

Sites sourced from `urls` will get scraped, with the retrieved words displayed on
`localhost:1337/metrics`

## Issues
1. As of writing this README, the scraper runs an unholy amount of Goroutines.
However, most of the goroutines are stagnant
2. The scraper does not follow redirects, only finds href elements, 
3. It does not use ETag and If-None-Match headers, nor does it use If-Modified-Since
(the cache isn't persistent and there is no way to invalidate it)  