package main

import (
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	_ "github.com/go-sql-driver/mysql"
)

type ScrapeResult struct {
	title    string
	text     string
	ImageURL string
}

type Parser interface {
	ParsePage(*goquery.Document) ScrapeResult
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

func crawlPage(baseURL, targetURL string, token chan struct{}) []string {

	token <- struct{}{}
	resp, _ := getRequest(targetURL)
	<-token

	doc, _ := goquery.NewDocumentFromResponse(resp)
	links := extractLinks(doc)
	foundUrls := resolveRelative(baseURL, links)

	return foundUrls
}

func parseStartURL(u string) string {
	parsed, _ := url.Parse(u)
	return fmt.Sprintf("%s://%s", parsed.Scheme, parsed.Host)
}

func targetpage(startURL string, concurrency int) []string {
	var foundLinks []string
	worklist := make(chan []string)
	var n int
	n++
	var tokens = make(chan struct{}, concurrency)
	go func() { worklist <- []string{startURL} }()
	seen := make(map[string]bool)
	baseDomain := parseStartURL(startURL)

	for i := 0; i < 3; i++ {
		list := <-worklist
		for _, link := range list {
			if !seen[link] {
				seen[link] = true
				n++
				go func(baseDomain, link string, token chan struct{}) {
					foundLinks = crawlPage(baseDomain, link, token)
					if foundLinks != nil {
						worklist <- foundLinks
					}
				}(baseDomain, link, tokens)
			}
		}
	}
	return foundLinks
}

func crawl(startURL string) []string {
	conn, err := sql.Open("mysql", "root:qordls7410@tcp(localhost:3306)/WIKI")
	target := (targetpage(startURL, 2))
	b := ScrapeResult{}
	var image []string
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	for i := 0; i < 5; i++ {
		resp, err := http.Get(target[i])

		if err != nil {
			panic(err)
		}
		defer resp.Body.Close()

		doc, err := goquery.NewDocumentFromReader(resp.Body)

		b.title = doc.Find("h1.firstHeading").Text()

		doc.Find("div.mw-parser-output img").Each(func(i int, s *goquery.Selection) {
			b.ImageURL, _ = s.Attr("src")
			image = append(image, b.ImageURL)
		})
		doc.Find("div.mw-parser-output p").Each(func(i int, s *goquery.Selection) {
			b.text += s.Text()
		})
		conn.Exec("insert into wiki_data(Title,Text,imageurl)value(?,?,?)", b.title, b.text, b.ImageURL)
		b.text = " "
	}
	return image
}

func crawlImage(startURL string) error {
	Image := crawl(startURL)
	var ImageURL []string
	for _, a := range Image {
		ImageURL = append(ImageURL, "https:"+a)
	}

	for i := 0; i < 5; i++ {
		resp, err := http.Get(ImageURL[i])

		if err != nil {
			return err
		}

		defer resp.Body.Close()

		out, err := os.Create(strconv.Itoa(i) + ".jpg")

		if err != nil {
			return err
		}

		defer out.Close()

		_, err = io.Copy(out, resp.Body)

		return err
	}
	return nil
}

func main() {
	crawlImage("https://ko.wikipedia.org/wiki/")

}
