package scraper

import (
	"bytes"
	"errors"
	"github.com/tschuyebuhl/scraper/cache"
	"github.com/tschuyebuhl/scraper/data"
	"golang.org/x/net/html"
	"io"
	"log/slog"
	"net/url"
	"regexp"
	"strings"
)

var (
	TooDeep    = errors.New("depth is too high (maximum depth is hardcoded to 2)")
	BadRequest = errors.New("error during get")
	BadBody    = errors.New("cannot parse body")
)

const MaxDepth = 2

/*
Scrape - a high level overview: consume a token, defer release,
check if depth is okay, check cache, "lock" url in cache, perform a GET, parse body, send results and queue more jobs
*/
func Scrape(
	urlStr string,
	c cache.Cache,
	resultsChan chan<- data.PageData,
	depth int,
	requester data.HTTPRequester,
	sem chan struct{},
) error {
	slog.Info("scraping: ", "urlStr", urlStr, "depth", depth)

	sem <- struct{}{}
	defer func() {
		slog.Info("releasing semaphore token")
		<-sem
	}()

	if depth > MaxDepth {
		slog.Error("too deep, stopping scrape")
		return TooDeep
	}

	if page, found := c.Get(urlStr); found {
		if page.CurrentlyScraping {
			slog.Info("URL is currently being scraped", "pageURL", page.URL)
			return nil
		}
		slog.Info("URL is already in cache: ", "url", page.URL)
		resultsChan <- *page
		return nil
	}

	c.Put(&data.PageData{
		URL:               urlStr,
		CurrentlyScraping: true,
	})

	resp, err := requester.Get(urlStr)
	if err != nil {
		return BadRequest
	}

	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return BadBody
	}
	// this can get noisy!
	//slog.Debug("body: ", "response", string(bodyBytes))

	extractedText := extractText(bytes.NewReader(bodyBytes))
	frequentWords := countWords(extractedText)

	resultsChan <- data.PageData{URL: urlStr, WordFrequency: frequentWords}

	links := FindLinks(bytes.NewReader(bodyBytes), urlStr)
	for _, link := range links {
		go func(link string) {
			if _, found := c.Get(link); !found {
				if depth+1 <= MaxDepth { //dumb failsafe but works I guess?
					slog.Info("adding to task list", "depth", depth+1, "url", link)
					_ = Scrape(link, c, resultsChan, depth+1, requester, sem)
				}
			}
		}(link)
	}

	realData := &data.PageData{
		URL:               urlStr,
		WordFrequency:     frequentWords,
		CurrentlyScraping: false,
	}
	c.Put(realData)

	return nil
}

func extractText(r io.Reader) string {
	z := html.NewTokenizer(r)
	var textContent strings.Builder

	for {
		t := z.Next()

		switch t {
		case html.ErrorToken:
			return textContent.String()
		case html.TextToken:
			textContent.WriteString(string(z.Text()))
		}
	}
}

func countWords(text string) map[string]int {
	wordCounts := make(map[string]int)
	words := strings.Fields(text)

	for _, word := range words {
		word = strings.ToLower(word)
		sanitized := SanitizeWord(word)
		if IsValidWord(sanitized) {
			wordCounts[sanitized]++
		}
	}
	return wordCounts
}

var validWord = regexp.MustCompile(`^[a-zA-Z]+$`)

func SanitizeWord(word string) string {
	return strings.ToLower(strings.TrimSpace(word))
}

func IsValidWord(word string) bool {
	return validWord.MatchString(word)
}

func FindLinks(body io.Reader, baseURL string) []string {
	var links []string
	doc, err := html.Parse(body)
	if err != nil {
		return links
	}

	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, a := range n.Attr {
				if a.Key == "href" {
					base, err := url.Parse(baseURL)
					if err != nil {
						continue
					}
					link, err := base.Parse(a.Val)
					if err == nil {
						links = append(links, link.String())
					}
					break
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)
	return links
}
