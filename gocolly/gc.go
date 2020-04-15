package main

import (
	"fmt"

	"github.com/gocolly/colly"
)

type data struct{
	text string
	imgURL string
}

func main() {
	// Instantiate default collector
	c := colly.NewCollector(
		colly.AllowedDomains("ko.wikipedia.org"),
		colly.MaxDepth(1),
	)

	// On every a element which has href attribute call callback
	c.OnHTML("div", func(e *colly.HTMLElement) {
		link := e.Attr("span")

		
		fmt.Printf("Link found: %q -> %s\n", e.Text, link)
		
		c.Visit(e.Request.AbsoluteURL(link))

	})

	
	c.Visit("https://ko.wikipedia.org/")
}