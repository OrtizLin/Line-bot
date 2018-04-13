package db

import (
	"encoding/json"
	"fmt"
	"github.com/utahta/go-linenotify"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"log"
	"net/http"
	"os"
	"strconv"
)

type User struct {
	UserToken string
}

type Article struct {
	Title     string
	LikeCount int
	Link      string
	Date      string
	ImageLink string
}

func SaveToken(token string) bool {

	session, errs := mgo.Dial(os.Getenv("DBURL"))
	if errs != nil {
		panic(errs)
	}
	defer session.Close()
	collect := session.DB("xtest").C("tokendb")
	user := User{}
	user.UserToken = token
	errs = collect.Insert(&User{user.UserToken})
	if errs != nil {
		log.Fatal(errs)
		return false
	} else {
		connect := linenotify.New()
		connect.NotifyWithImageURL(user.UserToken, "恭喜您已與表特爆報連動 , 若表特版有精彩文章將會立即通知您。", "https://image.famitsu.hk/201712/47dec32c774c3fd60deb142192fcee93_m.jpg", "https://image.famitsu.hk/201712/47dec32c774c3fd60deb142192fcee93_m.jpg")
		return true
	}

}

func InsertArticle(title, likeCount, link, date, imageLink, likeCountString string) {
	session, errs := mgo.Dial(os.Getenv("DBURL"))
	if errs != nil {
		panic(errs)
	}
	defer session.Close()
	c := session.DB("xtest").C("xtest")
	c2 := session.DB("xtest").C("alreadysent")
	c3 := session.DB("xtest").C("tokendb")
	errs = c.Insert(&Article{title, likeCount, link, date, imageLink, likeCountString})
	if errs != nil {
		log.Fatal(errs)
	} else {
		if likeCountString == "爆" {
			result := Article{}
			err := c2.Find(bson.M{"link": link}).One(&result_two) //check if article already send
			if err != nil {
				err3 := c2.Insert(&Article{title, likeCount, link, date, imageLink, likeCountString})
				if err3 != nil {
					log.Fatal(err3)
				}

				users := User{}
				iter := c3.Find(nil).Iter()
				for iter.Next(&users) {
					connect := linenotify.New()
					content := "\n" + link
					connect.NotifyWithImageURL(users.UserToken, content, imageLink, imageLink)
				}

			}

		}
	}
}

func SearchArticle(message string) (article []Article) {
	var articles []Article
	session, errs := mgo.Dial(os.Getenv("DBURL"))
	if errs != nil {
		panic(errs)
	}
	defer session.Close()
	c := session.DB("xtest").C("xtest")
	result := Article{}
	iter := c2.Find(bson.M{"title": bson.M{"$regex": message}}).Iter()
	for iter.Next(&result) {
		articles = append(articles, result)
	}
	return articles
}

func RemoveALL() {
	session, errs := mgo.Dial(os.Getenv("DBURL"))
	if errs != nil {
		panic(errs)
	}
	defer session.Close()
	c := session.DB("xtest").C("xtest")
	//Clean DB
	c.RemoveAll(nil)
}
