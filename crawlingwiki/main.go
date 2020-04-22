package main

import (
	"bytes"
	"database/sql"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	_ "github.com/go-sql-driver/mysql"
)

const (
	S3_REGION = "ap-northeast-2"
	S3_BUCKET = "gowiki"
)

type ScrapeResult struct {
	title    string
	text     string
	ImageURL []string
}

func getRequest(url string) (*http.Response, error) { //url 에 헤더를 추가하여 컴퓨터가아님을 우회하는 방법
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
	noduplicate := doc.Find("div#bodyContent")
	keys := make(map[string]bool)
	dupleUrl := []string{}

	if doc != nil {
		noduplicate.Find("a").Each(func(i int, s *goquery.Selection) {
			res, _ := s.Attr("href")
			if strings.Contains(res, ":") == false && strings.Contains(res, "/wiki/") == true {
				dupleUrl = append(foundUrls, res)
				for _, value := range dupleUrl {
					if _, saveValue := keys[value]; !saveValue {
						keys[value] = true
						foundUrls = append(foundUrls, value)
					}
				}
			}
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

	for ; n > 0; n-- {
		list := <-worklist
		fmt.Println(list)
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

func crawl(startURL string, wg *sync.WaitGroup) {

	conn, err := sql.Open("mysql", "root:qordls7410@tcp(localhost:3306)/WIKI")
	target := (targetpage(startURL, 5))
	b := ScrapeResult{}

	if err != nil {
		os.Exit(1)
	}
	for i := 0; i < len(target); i++ {
		resp, err := http.Get(target[i])

		if err != nil {
			panic(err)
		}
		defer resp.Body.Close()

		doc, err := goquery.NewDocumentFromReader(resp.Body)

		b.title = doc.Find("h1.firstHeading").Text()
		doc.Find("div.mw-parser-output p").Each(func(i int, s *goquery.Selection) {
			b.text += s.Text()
		})
		conn.Exec("insert into wiki_data(Title,Text) value(?,?)", b.title, b.text)
		b.text = " "

	}
	fmt.Print(target)
}

func FindImageurl(startURL string) []string {
	target := (targetpage(startURL, 5))
	b := ScrapeResult{}
	for i := 0; i < len(target); i++ {
		resp, err := http.Get(target[i])

		if err != nil {
			panic(err)
		}
		defer resp.Body.Close()

		doc, err := goquery.NewDocumentFromReader(resp.Body)

		doc.Find("img.thumbimage").Each(func(i int, s *goquery.Selection) {
			image, ok := s.Attr("src")
			if ok {
				b.ImageURL = append(b.ImageURL, image)
			} else {
				b.ImageURL = append(b.ImageURL, "//upload.wikimedia.org/wikipedia/commons/thumb/4/4a/Commons-logo.svg/30px-Commons-logo.svg.png")
			}
		})
	}
	return b.ImageURL
}

func ImageDownload(startURL string, wg *sync.WaitGroup) error {
	Image := FindImageurl(startURL)
	var ImageURL []string
	for _, a := range Image {
		ImageURL = append(ImageURL, "https:"+a)
	}

	s, err := session.NewSession(&aws.Config{Region: aws.String(S3_REGION)})

	if err != nil {
		log.Fatal(err)
	}
	for i := 0; i < len(ImageURL); i++ {
		resp, err := http.Get(ImageURL[i])

		if err != nil {
			return err
		}

		defer resp.Body.Close()

		out, err := os.Create("img/" + strconv.Itoa(i) + ".jpg")

		if err != nil {
			return err
		}

		defer out.Close()

		_, err = io.Copy(out, resp.Body)

		err = AddFileTOS3(s, "img/"+strconv.Itoa(i)+".jpg")

		if err != nil {
			log.Fatal(err)
		}
	}

	return nil
}

func AddFileTOS3(s *session.Session, fileDir string) error {

	file, err := os.Open(fileDir)
	if err != nil {
		return err
	}

	defer file.Close()

	fileInfo, _ := file.Stat()
	var size int64 = fileInfo.Size()
	buffer := make([]byte, size)
	file.Read(buffer)

	_, err = s3.New(s).PutObject(&s3.PutObjectInput{
		Bucket:               aws.String(S3_BUCKET),
		Key:                  aws.String(fileDir),
		ACL:                  aws.String("private"),
		Body:                 bytes.NewReader(buffer),
		ContentLength:        aws.Int64(size),
		ContentType:          aws.String("http.DetectContentType(buffer)"),
		ContentDisposition:   aws.String("attachment"),
		ServerSideEncryption: aws.String("AES256"),
	})
	return err
}

func main() {
	var wg sync.WaitGroup
	wg.Add(2)

	go crawl("https://ko.wikipedia.org/wiki/", &wg)

	go ImageDownload("https://ko.wikipedia.org/wiki/", &wg)

	wg.Wait()

}
