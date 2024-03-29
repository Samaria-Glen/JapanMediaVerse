package main

import (
	"log"
	"os"
	"strconv"

	"github.com/yomorun/yomo"
	"github.com/yomorun/yomo/serverless"
)

type noiseData struct {
	Noise float32 `json:"noise"` // Noise value
	Time  int64   `json:"time"`  // Timestamp (ms)
	From  string  `json:"from"`  // Source IP
}

func main() {
	addr := "localhost:9000"
	if v := os.Getenv("YOMO_ADDR"); v != "" {
		addr = v
	}
	sfn := yomo.NewStreamFunction(
		"sfn-1",
		addr,
	)
	sfn.SetObserveDataTags(0x33)
	defer sfn.Close()

	// set handler
	sfn.SetHandler(handler)
	// start
	err := sfn.Connect()
	if err != nil {
		log.Printf("[sfn-1] connect err=%v", err)
		os.Exit(1)
	}
	// set the error handler function when server error occurs
	sfn.SetErrorHandler(func(err error) {
		log.Printf("[sfn-1] receive server error: %v", err)
		sfn.Close()
		os.Exit(1)
	})

	sfn.Wait()
}

func handler(ctx serverless.Context) {
	// got
	data := ctx.Data()
	noise, err := strconv.ParseFloat(string(data), 10)
	if err != nil {
		log.Printf("[sfn-1] got err=%v", err)
		return
	}
	// result
	result := int(noise)
	resultTag := uint32(0x34)
	log.Printf(
		"[sfn-1] got: tag=%#x, data=%v, return: tag=%#x, data=%v\n",
		ctx.Tag(), noise,
		resultTag,
		result,
	)
	output := []byte(strconv.Itoa(result))
	err = ctx.Write(resultTag, output)
}
