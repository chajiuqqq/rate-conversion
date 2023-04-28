package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"time"
)

type QueryResponse struct {
	Asset_id_base  string      `json:"asset_id_base"`
	Asset_id_quote string      `json:"asset_id_quote"`
	Rate           json.Number `json:"rate"`
	Time           string      `json:"time"`
}

type RateBody struct {
	createTime time.Time
	value      *QueryResponse
}

type ConvertService struct {
	// key: `from-to` value:Rate
	cache   map[string]*RateBody
	baseUrl string
	client  *http.Client
	header  http.Header
	expire  time.Duration
	rwLock  sync.RWMutex
}

func NewConvertService(key string, expire time.Duration) *ConvertService {
	header := http.Header{}
	header.Add("X-CoinAPI-Key", key)

	service := &ConvertService{
		cache:   map[string]*RateBody{},
		baseUrl: "https://rest.coinapi.io/v1/exchangerate",
		client:  &http.Client{},
		header:  header,
		expire:  expire,
	}
	ticker := time.NewTicker(time.Second)
	go func() {
		for {
			select {
			case <-ticker.C:
				for k, v := range service.cache {
					if time.Since(v.createTime) > service.expire {
						service.rwLock.Lock()
						delete(service.cache, k)
						service.rwLock.Unlock()
					}
				}
			}
		}

	}()

	return service
}

func (c *ConvertService) getCachedResult(from, to string) (*QueryResponse, bool) {
	key := from + "-" + to
	c.rwLock.RLock()
	defer c.rwLock.RUnlock()
	if v, ok := c.cache[key]; ok {
		if time.Since(v.createTime) < time.Minute*10 { // 检查缓存是否过期
			return v.value, true
		} else {
			delete(c.cache, key) // 如果缓存过期则删除缓存
		}
	}
	return nil, false
}

func (c *ConvertService) cacheResult(from, to string, result *QueryResponse) {
	key := from + "-" + to
	c.rwLock.Lock()
	defer c.rwLock.Unlock()
	c.cache[key] = &RateBody{
		value:      result,
		createTime: time.Now(),
	}
}

func (c *ConvertService) Convert(from, to string) (*QueryResponse, error) {
	if v, ok := c.getCachedResult(from, to); ok {
		log.Printf("From %s to %s Use cached", from, to)
		return v, nil
	}

	url := fmt.Sprintf("%s/%s/%s", c.baseUrl, from, to)
	r, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	r.Header = c.header

	resp, err := c.client.Do(r)
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, errors.New(string(body))
	}

	myres := &QueryResponse{}
	err = json.Unmarshal(body, myres)
	if err != nil {
		return nil, err
	}

	c.cacheResult(from, to, myres)
	log.Printf("From %s to %s Not Use cached", from, to)
	return myres, nil
}

func (c *ConvertService) PrintCache() {
	for k, v := range c.cache {
		expire := c.expire.Seconds() - time.Since(v.createTime).Seconds()
		log.Println(k, v.value.Rate, expire, "s")
	}
}
