package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/anvh2/sse/sse"
	"github.com/go-redis/redis"
)

func main() {
	broker := sse.NewServer()

	go func() {
		for {
			time.Sleep(2 * time.Second)
			eventString := fmt.Sprintf("the time is %v", time.Now())
			log.Println("Sending event")
			
			redisCli := redis.NewClient(&redis.Options{
				Addr: "localhost:6379",
			})

			if err := redisCli.Ping().Err(); err != nil {
				panic(err)
			}

			redisCli.Publish("foo", eventString)
		}
	}()

	err := http.ListenAndServe("localhost:8000", broker)
	if err != nil {
		log.Fatal("HTTP server error: ", err)
	}
}
