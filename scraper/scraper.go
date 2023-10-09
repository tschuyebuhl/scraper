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
	"sync"
)

var (
	TooDeep = errors.New("depth is too high (maximum depth is hardcoded to 1)")
	BadUrl  = errors.New("cannot parse url")
	BadBody = errors.New("cannot parse body")
)

const MaxDepth = 1

func Scrape(
	urlStr string,
	c cache.Cache,
	//resultsChan chan<- map[string]int,
	resultsChan chan<- data.PageData,
	wg *sync.WaitGroup,
	depth int,
	sem chan struct{},
	taskChan chan<- string,
	requester data.HTTPRequester,
) error {
	slog.Info("scraping: ", "urlStr", urlStr, "depth", depth)

	sem <- struct{}{}
	defer func() {
		<-sem
		wg.Done()
	}()

	if depth > MaxDepth {
		return TooDeep
	}

	if page, found := c.Get(urlStr); found {
		if page.IsScraped {
			slog.Info("URL is currently being scraped", "pageURL", page.URL)
			return nil
		}
		slog.Info("URL is already in cache: ", "url", page.URL)
		resultsChan <- *page
		return nil
	}

	resp, err := requester.Get(urlStr)
	if err != nil {
		return BadUrl
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return BadBody
	}
	// this can get noisy!
	//slog.Info("body: ", "response", string(bodyBytes))

	extractedText := extractText(bytes.NewReader(bodyBytes))
	frequentWords := countWords(extractedText)

	resultsChan <- data.PageData{URL: urlStr, WordFrequency: frequentWords}

	links := FindLinks(bytes.NewReader(bodyBytes), urlStr)
	for _, link := range links {
		if _, found := c.Get(link); !found {
			wg.Add(1)
			go func(link string, nextDepth int) {
				taskChan <- link
			}(link, depth+1)
		}
	}
	realData := &data.PageData{
		URL:           urlStr,
		WordFrequency: frequentWords,
		IsScraped:     false,
	}
	c.Put(urlStr, realData)

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
	links := []string{}
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
