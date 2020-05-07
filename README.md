*#Golang crawler*

###require packages
"bytes" // 사진의 크기를 정해주기 위해사용

"database/sql" //sql구문을 사용하기 위해 사용

"fmt"

"io" //사진을 저장하기위해 사용

"log" //error 확인

"net/http" //http를 가져올때사용

"net/url" // url문자열 조작

"os" // 디렉토리 설정 

"strconv" // string convert 패키지

"strings" // regexp 대신 사용 regexp는 실행시간이 길어질수도 있는 단점이 있다. 정규식 regexp사용 피하기 (go언어 실전테크닉 참조)

"github.com/PuerkitoBio/goquery" //goquery html을 파싱

"github.com/aws/aws-sdk-go/aws"     //aws 관련

"github.com/aws/aws-sdk-go/aws/session" //aws관련

"github.com/aws/aws-sdk-go/service/s3" //aws관련

_ "github.com/go-sql-driver/mysql" //golang 과 mysql을 연동



##install 

`go get github.com/PuerkitoBio/goquery`

`go getgithub.com/go-sql-driver/mysql`

`go get github.com/aws/aws-sdk-go/...`

>  mysql db는 제 로컬로 되어있어서 바꿔야합니다. -주석 처리 해놓았습니다. 바꾸어야 할부분

>  aws key 와 bucket name 이 제 개발서버로 되어있어서 바꾸어야 합니다. -주석 처리 해놓았습니다. 바꾸어야 할부분

#시작

`go run main.go`

