package main

import (
	"/Users/baeg-injun/go/src/golang-wiki-crawler/crawling"
	"github.com/PuerkitoBio/goquery"
)

type DummyParser struct {
}

func (x DummyParser) ParsePage(doc *goquery.Document) crawling.ScraptResult {
	data := crawling.ScraptResult{}
	data.Title = doc.Find("title").First().Text()
	data.H1 = doc.Find("h1").First().Text()
	return crawling.ScraptResult{}

}

func main() {
	d := DummyParser{}
	crawling.Crawl("https://ko.wikipedia.org/", d, 10)
}
