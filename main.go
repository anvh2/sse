package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/anvh2/sse/sse"
)

func main() {
	broker := sse.NewServer()

	go func() {
		for {
			time.Sleep(2 * time.Second)
			eventString := fmt.Sprintf("the time is %v", time.Now())
			log.Println("Sending event")
			broker.Notifier <- []byte(eventString)
		}
	}()

	err := http.ListenAndServe("localhost:8000", broker)
	if err != nil {
		log.Fatal("HTTP server error: ", err)
	}
}
