package main

import (
	"fmt"
	"log"
	"time"

	sse "github.com/anvh2/sse/server"
	"github.com/go-redis/redis"
)

func main() {
	broker := sse.NewServer()
	err := broker.Run()
	if err != nil {
		log.Fatal("HTTP server error: ", err)
	}

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
}
