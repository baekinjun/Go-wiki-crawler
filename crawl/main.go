package main

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

type ScrapeResult struct {
	ImageURL []string
	title    []string
	text     []string
}

func getRequest(url string) (*http.Response, error) {
	client := &http.Client{}

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)")

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func extractLinks(doc *goquery.Document) []string {
	foundUrls := []string{}
	if doc != nil {
		doc.Find("a").Each(func(i int, s *goquery.Selection) {
			res, _ := s.Attr("href")
			foundUrls = append(foundUrls, res)
		})
		return foundUrls
	}
	return foundUrls
}

func resolveRelative(baseURL string, hrefs []string) []string {
	internalUrls := []string{}

	for _, href := range hrefs {
		if strings.HasPrefix(href, baseURL) {
			internalUrls = append(internalUrls, href)
		}

		if strings.HasPrefix(href, "/") {
			resolvedURL := fmt.Sprintf("%s%s", baseURL, href)
			internalUrls = append(internalUrls, resolvedURL)
		}
	}

	return internalUrls
}

func pagesort(baseurl, targetUrl string) []string {
	resp, _ := getRequest(targetUrl)
	doc, _ := goquery.NewDocumentFromResponse(resp)
	links := extractLinks(doc)
	foundurls := resolveRelative(baseurl, links)

	return foundurls

}

func parseStartURL(u string) string {
	parsed, _ := url.Parse(u)
	return fmt.Sprintf("%s://%s", parsed.Scheme, parsed.Host)
}

func targetpage(starturl string) []string {
	results := []ScrapeResult{}
	basedomain := parseStartURL(starturl)
	foundurls := pagesort(basedomain, "https://ko.wikipedia.org/wiki/위키백과:오늘의_역사")
	results = append(results)
	return foundurls
}

func main() {
	// 테스트중 테스트후 func crawl으로 만들기
	a := (targetpage("https://ko.wikipedia.org/wiki/"))
	b := ScrapeResult{}
	var image string
	for i := 0; i < 5; i++ {
		resp, err := http.Get(a[i])
		fmt.Println(a[i])
		if err != nil {
			panic(err)
		}
		defer resp.Body.Close()

		doc, err := goquery.NewDocumentFromReader(resp.Body)

		b.title = append(b.title, doc.Find("title").Text())
		doc.Find("div.mw-parser-output").Each(func(i int, s *goquery.Selection) {
			b.text = append(b.text, s.Text())
		})
		doc.Find("div.mw-parser-output img").Each(func(i int, s *goquery.Selection) {
			image, _ = s.Attr("src")
			b.ImageURL = append(b.ImageURL, image)
		})

	}
	fmt.Println(b.title)
	fmt.Println(b.text)
	fmt.Println(b.ImageURL)

}
