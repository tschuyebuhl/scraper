package scraper

import (
	"bytes"
	"errors"
	"github.com/tschuyebuhl/scraper/cache"
	"github.com/tschuyebuhl/scraper/data"
	"golang.org/x/net/html"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
)

var (
	TooDeep = errors.New("depth is too high (maximum depth is )")
	BadUrl  = errors.New("cannot parse url")
	BadBody = errors.New("cannot parse body")
)

func isTextContainer(tag string) bool {
	safeTags := []string{"p", "h1", "h2", "h3", "h4", "h5", "h6", "li"}
	for _, t := range safeTags {
		if tag == t {
			return true
		}
	}
	return false
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

const MaxDepth = 1

func Scrape(urlStr string, c cache.Cache, resultsChan chan<- map[string]int,
	wg *sync.WaitGroup, depth int, sem chan struct{},
	taskChan chan<- string) error {
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
			return nil // early exit
		}
		slog.Info("URL is already in cache: ", page.URL)
		resultsChan <- page.WordFrequency
		return nil
	}

	placeholder := &data.PageData{
		URL:       urlStr,
		IsScraped: true,
	}
	c.Put(urlStr, placeholder)

	resp, err := http.Get(urlStr)
	if err != nil {
		slog.Error("error in get: %s", err)
		return BadUrl
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return BadBody
	}

	extractedText := extractText(bytes.NewReader(bodyBytes))
	frequentWords := countWords(extractedText)

	resultsChan <- frequentWords
	links := FindLinks(bytes.NewReader(bodyBytes), urlStr)
	for _, link := range links {
		if _, found := c.Get(link); !found {
			wg.Add(1)
			taskChan <- link
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

// Find links
/*links := FindLinks(bytes.NewReader(bodyBytes), urlStr)
  for _, link := range links {
  	wg.Add(1)
  	link := link //was this fixed in 1.22?
  	go func() {
  		_ = Scrape(link, c, resultsChan, wg, depth+1)
  	}()
  }

*/
