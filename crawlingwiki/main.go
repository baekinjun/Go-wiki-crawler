package main

import (
	"bytes"        // 사진의 크기를 정해주기 위해사용
	"database/sql" //sql구문을 사용하기 위해 사용
	"fmt"
	"io"       //사진을 저장하기위해 사용
	"log"      //error 확인
	"net/http" //http를 가져올때 사용
	"net/url"  // url문자열 조작
	"os"       // 디렉토리 설정
	"strconv"  // string convert 패키지
	"strings"  // regexp 대신 사용 regexp는 실행시간이 길어질수도 있는 단점이 있다. (go언어 실전테크닉 참조)

	"github.com/PuerkitoBio/goquery"        //goquery html을 파싱
	"github.com/aws/aws-sdk-go/aws"         //aws 관련
	"github.com/aws/aws-sdk-go/aws/session" //aws관련
	"github.com/aws/aws-sdk-go/service/s3"  //aws관련
	_ "github.com/go-sql-driver/mysql"      //golang 과 mysql을 연동
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

func extractLinks(doc *goquery.Document) []string { //페이지 내의 baseurl에 덧붙힐 extra주소 수집
	foundUrls := []string{}
	noduplicate := doc.Find("div#bodyContent")
	keys := make(map[string]bool)
	dupleUrl := []string{}

	if doc != nil {
		noduplicate.Find("a").Each(func(i int, s *goquery.Selection) {
			res, _ := s.Attr("href")
			if strings.Contains(res, ":") == false && strings.Contains(res, "/wiki/") == true { // 위키피디아의 html분석결과 검색을 통한것들은 : 이포함되어 있지않고 /wiki/로 시작한다.
				dupleUrl = append(foundUrls, res)
				for _, value := range dupleUrl {
					if _, saveValue := keys[value]; !saveValue { //56~60 번째줄은 중복된것을 제거 하는것
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

func resolveRelative(baseURL string, hrefs []string) []string { //문자열을 재조합하여 원하는 url을 가지고 온다.
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

func crawlPage(baseURL, targetURL string, token chan struct{}) []string { //baseurl과 target url을 합친다.

	token <- struct{}{}
	resp, _ := getRequest(targetURL)
	<-token

	doc, _ := goquery.NewDocumentFromResponse(resp)
	links := extractLinks(doc)
	foundUrls := resolveRelative(baseURL, links)

	return foundUrls
}

func parseStartURL(u string) string { //net/url 패키지 기능의로 url의 구문을 분석하여 starturl을 얻는다.
	parsed, _ := url.Parse(u)
	return fmt.Sprintf("%s://%s", parsed.Scheme, parsed.Host)
}

func targetpage(startURL string, concurrency int) []string { // crawlpage를 계속 돌려(url이 소진시까지 ) targeturl을 가져온다.
	var foundLinks []string
	worklist := make(chan []string)
	var n int
	n++
	var tokens = make(chan struct{}, concurrency)
	go func() { worklist <- []string{startURL} }()
	seen := make(map[string]bool)
	baseDomain := parseStartURL(startURL)
	// ; n > 0; n-- (<-모든 url을 가지고올때  for문에 대입) 모든 url이 소진시 멈춘다.
	for i := 0; i < 10; i++ {
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

func crawl(startURL string) ([]string, []string) { // target url의 원하는데이터를 정제한후에  디비에 저장
	conn, err := sql.Open("mysql", "root:qordls7410@tcp(localhost:3306)/WIKI")
	target := (targetpage(startURL, 5))
	var connect []string
	var connectDB string
	b := ScrapeResult{}
	if err != nil {
		os.Exit(1)
	}
	fmt.Println(len(target))
	for i := 0; i < len(target); i++ {
		resp, err := http.Get(target[i])
		if err != nil {
			panic(err)
		}
		defer resp.Body.Close()

		doc, err := goquery.NewDocumentFromReader(resp.Body)

		b.title = doc.Find("h1.firstHeading").Text()
		doc.Find("img.thumbimage").Each(func(i int, s *goquery.Selection) {
			image, ok := s.Attr("src")
			if ok {
				b.ImageURL = append(b.ImageURL, image)
				connectDB += b.title + strconv.Itoa(i) + ","
				connect = append(connect, b.title+strconv.Itoa(i))
			} else {
				b.ImageURL = append(b.ImageURL, "//upload.wikimedia.org/wikipedia/commons/thumb/4/4a/Commons-logo.svg/30px-Commons-logo.svg.png")
				connectDB = "No Image"
			}
		})
		doc.Find("div.mw-parser-output p").Each(func(i int, s *goquery.Selection) {
			b.text += s.Text()
		})
		conn.Exec("insert into wiki_data(Title,Text,connectimage) value(?,?,?)", b.title, b.text, connectDB)
		b.text = " "
		connectDB = " "

	}

	return b.ImageURL, connect
}

func ImageDownloadandcrawl(startURL string) error { // imageurl을 받아서 image를 저장후 aws에 저장
	Image, connectname := crawl(startURL)
	var ImageURL []string
	for _, a := range Image {
		ImageURL = append(ImageURL, "https:"+a)
	}
	fmt.Println(len(ImageURL))

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

		out, err := os.Create("img/" + connectname[i] + ".jpg")

		if err != nil {
			return err
		}

		defer out.Close()

		_, err = io.Copy(out, resp.Body)

		err = AddFileTOS3(s, "img/"+connectname[i]+".jpg")

		if err != nil {
			log.Fatal(err)
		}
	}

	return nil
}

func AddFileTOS3(s *session.Session, fileDir string) error { //aws configure 를 통해 너드팩토리 서버로 접속 해야됨 aws에 저장

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
	ImageDownloadandcrawl("https://ko.wikipedia.org/wiki/축구")
}
