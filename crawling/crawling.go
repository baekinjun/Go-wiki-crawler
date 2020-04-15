package crawling

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

type ScraptResult struct {
	URL   string
	Title string
	H1    string
}

type Parser interface {
	ParsePage(*goquery.Document) ScraptResult
}

func getRequest(url string) (*http.Response, error) {
	Client := &http.Client{}

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)")

	res, err := Client.Do(req)

	if err != nil {
		return nil, err
	}

	return res, nil
}





func crawlPage(baseURL, targetUrl string, parser Parser, token chan struct{}) ([]string, ScraptResult) {
	token <- struct{}{}
	fmt.Println("Requesting:", targetUrl)
	resp, _ := getRequest(targetUrl)
	<-token

	doc, _ := goquery.NewDocumentFromResponse(resp)
	pageResults := parser.ParsePage(doc)
	links := extractLinks(doc)
	// foundUrls := resolveRelative(baseURL, links)

	return links, pageResults
}

func parseStartURL(u string) string {
	parsed, _ := url.Parse(u)
	return fmt.Sprintf("%s // %s", parsed.Scheme, parsed.Host)
}

func Crawl(startURL string, parser Parser, concurenccy int) []ScraptResult {
	results := []ScraptResult{}
	worklist := make(chan []string)
	var n int
	n++
	var tokens = make(chan struct{}, concurenccy)
	go func() { worklist <- []string{startURL} }()
	seen := make(map[string]bool)
	baseDomain := parseStartURL(startURL)

	go func(baseDomain, link string, parser Parser, token chan struct{}) {
		pageResults := crawlPage(baseDomain, link, parser, token)
		results = append(results, pageResults)
				}(baseDomain, link, parser, tokens)

	return results
}
