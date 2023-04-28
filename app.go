package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

var URL = "https://rest.coinapi.io/v1/exchangerate/"

type MyResponse struct {
	Asset_id_base  string
	Asset_id_quote string
	Rate           float64
}

type CacheResponse struct {
	createTime time.Time
	value      *MyResponse
}

type MyCache struct {
	amount map[string]*CacheResponse
}

var mycache = &MyCache{amount: make(map[string]*CacheResponse)}

func main() {
	route := gin.Default()
	// ctx,cancel := context.WithCancel(context.Background())

	route.GET("/price/:currency", func(ctx *gin.Context) {
		target := ctx.Param("currency")
		if target == "" {
			ctx.JSON(500, gin.H{"msg": "empty param"})
		}
		target = strings.ToUpper(target)
		myres, err := getPrice(target)
		if err != nil {
			ctx.JSON(500, gin.H{"msg": err.Error()})
		}
		ctx.JSON(200, myres)
	})
	go func() {
		select {
		// case <-ctx.Done():
		// 	return
		case <-time.Tick(time.Second * 10):
			for k, v := range mycache.amount {
				if time.Now().Sub(v.createTime) > time.Second*10 {
					delete(mycache.amount, k)
				}
			}
		}
	}()
	route.Run(":8080")
}

func getPrice(target string) (*MyResponse, error) {
	if v, ok := mycache.amount[target]; ok {
		log.Println("Use cached")
		return v.value, nil
	}
	url := URL + target + "/USD"
	client := &http.Client{}
	r, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	r.Header.Add("X-CoinAPI-Key", "B89898B1-1DFC-4D44-AB49-4D56856A3627")

	resp, _ := client.Do(r)
	body, _ := ioutil.ReadAll(resp.Body)
	log.Println(string(body))
	myres := &MyResponse{}
	err = json.Unmarshal(body, myres)
	if err != nil {
		return nil, err
	}

	mycache.amount[target] = &CacheResponse{
		value:      myres,
		createTime: time.Now(),
	}
	log.Println("Not Use cached")
	return myres, nil
}
