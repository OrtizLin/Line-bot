package main

import (
	"github.com/line/line-bot-sdk-go/linebot"
	"github.com/utahta/go-linenotify"
	"github.com/utahta/go-linenotify/auth"
	"github.com/utahta/go-linenotify/token"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

type Article struct {
	Title     string
	LikeCount int
	Link      string
	Date      string
	ImageLink string
}

type User struct {
	UserToken string
}

func main() {
	app, err := NewLineBot(
		os.Getenv("ChannelSecret"),
		os.Getenv("ChannelAccessToken"),
		os.Getenv("APP_BASE_URL"),
	)
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/wakeup", WakeUp)
	http.HandleFunc("/auth", Authorize)
	http.HandleFunc("/pushnotify", Token)
	http.HandleFunc("/callback", app.Callback)
	if err := http.ListenAndServe(":"+os.Getenv("PORT"), nil); err != nil {
		log.Fatal(err)
	}
}

// line_bot app
type LineBot struct {
	bot         *linebot.Client
	appBaseURL  string
	downloadDir string
}

// NewLineBot function
func NewLineBot(channelSecret, channelToken, appBaseURL string) (*LineBot, error) {
	bot, err := linebot.New(
		channelSecret,
		channelToken,
	)
	if err != nil {
		return nil, err
	}

	return &LineBot{
		bot:         bot,
		appBaseURL:  appBaseURL,
		downloadDir: "test",
	}, nil
}

//wake up heroku server
func WakeUp(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Hello World")
}

//auth func for line notify
func Authorize(w http.ResponseWriter, req *http.Request) {
	c, err := auth.New(os.Getenv("ClientID"), os.Getenv("APP_BASE_URL")+"pushnotify")
	if err != nil {
		fmt.Fprintf(w, "error:%v", err)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: "state", Value: c.State, Expires: time.Now().Add(60 * time.Second)})

	c.Redirect(w, req)
}

//auth token for line notify
func Token(w http.ResponseWriter, req *http.Request) {
	resp, err := auth.ParseRequest(req)
	if err != nil {
		fmt.Fprintf(w, "error:%v", err)
		return
	}
	state, err := req.Cookie("state")
	if err != nil {
		fmt.Fprintf(w, "error:%v", err)
		return
	}
	if resp.State != state.Value {
		fmt.Fprintf(w, "error:%v", err)
		return
	}

	c := token.New(os.Getenv("APP_BASE_URL")+"pushnotify", os.Getenv("ClientID"), os.Getenv("ClientSecret"))
	accessToken, err := c.GetAccessToken(resp.Code)
	if err != nil {
		fmt.Fprintf(w, "error:%v", err)
		return
	}
	session, errs := mgo.Dial(os.Getenv("DBURL"))
	if errs != nil {
		panic(errs)
	}
	defer session.Close()
	collect := session.DB("xtest").C("tokendb")
	user := User{}
	user.UserToken = accessToken
	errs = collect.Insert(&User{user.UserToken})
	if errs != nil {
		log.Fatal(errs)
	} else {
		connect := linenotify.New()
		connect.NotifyWithImageURL(user.UserToken, "恭喜您已與表特爆報連動 , 若表特版有精彩文章將會立即通知您。", "https://image.famitsu.hk/201712/47dec32c774c3fd60deb142192fcee93_m.jpg", "https://image.famitsu.hk/201712/47dec32c774c3fd60deb142192fcee93_m.jpg")
	}
	fmt.Fprint(w, resp)
	fmt.Fprintf(w, "LINE Notify 連動完成。\n 您將可以不定期收到 [PTT 表特版] 爆文通知。")
}
func (app *LineBot) Callback(w http.ResponseWriter, r *http.Request) {
	events, err := app.bot.ParseRequest(r)

	if err != nil {
		if err == linebot.ErrInvalidSignature {
			w.WriteHeader(400)
		} else {
			w.WriteHeader(500)
		}
		return
	}
	for _, event := range events {
		log.Printf("Got event %v", event)
		switch event.Type {
		case linebot.EventTypeMessage:
			switch message := event.Message.(type) {
			case *linebot.TextMessage:
				if err := app.handleText(message, event.ReplyToken, event.Source); err != nil {
					log.Print(err)
				}
			default:
				log.Printf("Unknown message: %v", message)
			}
		default:
			log.Printf("Unknown event: %v", event)
		}
	}
}

func (app *LineBot) handleText(message *linebot.TextMessage, replyToken string, source *linebot.EventSource) error {
	switch message.Text {
	case "tonygrr":
		log.Printf("Echo message to %s: %s", replyToken, message.Text)
		if _, err := app.bot.ReplyMessage(
			replyToken,
			linebot.NewTextMessage("http://www.jav777.cc/"),
		).Do(); err != nil {
			return err
		}
	default:

		session, errs := mgo.Dial(os.Getenv("DBURL"))
		if errs != nil {
			panic(errs)
		}
		defer session.Close()
		c := session.DB("xtest").C("xtest")
		result := Article{}
		var columns []*linebot.CarouselColumn
		iter := c.Find(bson.M{"title": bson.M{"$regex": message.Text}}).Iter()
		var index = 0
		for iter.Next(&result) {
			if index == 10 { //array of columns, max:10
				break
			}
			thumbnailImageUrl := result.ImageLink
			column := linebot.NewCarouselColumn(
				thumbnailImageUrl, result.Date, result.Title,
				linebot.NewURITemplateAction("點我查看更多", result.Link),
			)
			columns = append(columns, column)
			index++
		}
		//if serch result is null
		if index == 0 {
			log.Printf("Echo message to %s: %s", replyToken, message.Text)
			if _, err := app.bot.ReplyMessage(
				replyToken,
				linebot.NewTextMessage("抱歉！目前無相關文章"),
			).Do(); err != nil {
				return err
			}
		} else {
			//reply carousel message if search result exist
			template := linebot.NewCarouselTemplate(columns...)
			if _, err := app.bot.ReplyMessage(
				replyToken,
				linebot.NewTemplateMessage("正妹來囉！", template),
			).Do(); err != nil {
				return err
			}
		}
	}
	return nil

}
