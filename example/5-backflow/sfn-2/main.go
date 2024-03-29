package main

import (
	"log"
	"os"
	"strconv"

	"github.com/yomorun/yomo"
	"github.com/yomorun/yomo/serverless"
)

func main() {
	addr := "localhost:9000"
	if v := os.Getenv("YOMO_ADDR"); v != "" {
		addr = v
	}
	sfn := yomo.NewStreamFunction(
		"sfn-2",
		addr,
	)
	sfn.SetObserveDataTags(0x34)
	defer sfn.Close()

	// set handler
	sfn.SetHandler(handler)
	// start
	err := sfn.Connect()
	if err != nil {
		log.Printf("[sfn-2] connect err=%v", err)
		os.Exit(1)
	}
	// set the error handler function when server error occurs
	sfn.SetErrorHandler(func(err error) {
		log.Printf("[sfn-2] receive server error: %v", err)
		sfn.Close()
		os.Exit(1)
	})

	sfn.Wait()
}

func handler(ctx serverless.Context) {
	// got
	data := ctx.Data()
	noise, err := strconv.Atoi(string(data))
	if err != nil {
		log.Printf("[sfn-2] got err=%v", err)
		return
	}
	// result
	result := noise * 10
	log.Printf("[sfn-2] got: tag=0x34, data=%v, return: tag=0x35, data=%v", noise, result)

	ctx.Write(0x35, []byte(strconv.Itoa(result)))
}
