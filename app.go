package main

import (
	. "example/service"
	"flag"
	"log"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

var (
	key            = flag.String("key", "", "access key")
	convertService *ConvertService
)
const (
	CACHE_EXPIRE = time.Second*10
)

func init() {
	flag.Parse()
	if key == nil || *key == "" {
		log.Fatal("key couldn't be empty")
		return
	}
	convertService = NewConvertService(*key, CACHE_EXPIRE)
}

func main() {

	route := gin.Default()
	route.GET("/price/:currency", func(ctx *gin.Context) {
		currency := ctx.Param("currency")
		currency = strings.ToUpper(currency)
		res, err := convertService.Convert(currency, "USD")
		if err != nil {
			ctx.JSON(500, gin.H{"msg": "convert error", "error": err.Error()})
			return
		}
		// convertService.PrintCache()
		ctx.JSON(200, res)
	})
	route.Run(":8080")
}
