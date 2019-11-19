package sse

import (
	"fmt"
	"log"
	"net/http"

	"github.com/go-redis/redis"
)

// MessageChan -
type MessageChan []byte

// Broker holds open client  connections,
// listen for incoming events on it Notifier channel
// and broadcast event data to all registered connections
type Broker struct {
	// Events are pushed to this channel by the main events-gathering
	Notifier chan MessageChan

	// New client connections
	newClients chan chan MessageChan

	// Closed client connections
	closingClients chan chan MessageChan

	// Client connections registery
	clients map[chan MessageChan]bool

	// quitc
	quitc chan struct{}
}

// NewServer -
func NewServer() *Broker {
	return &Broker{
		Notifier:       make(chan MessageChan, 1),
		newClients:     make(chan chan MessageChan),
		closingClients: make(chan chan MessageChan),
		clients:        make(map[chan MessageChan]bool),
		quitc:          make(chan struct{}),
	}
}

// Run -
func (broker *Broker) Run() error {
	// setup SSE monitor
	go func() {
		for {
			select {
			case s := <-broker.newClients:
				broker.clients[s] = true
				log.Printf("Client added. %d registered clients", len(broker.clients))
			case s := <-broker.closingClients:
				delete(broker.clients, s)
				log.Printf("Removed client. %d registered clients", len(broker.clients))
			case event := <-broker.Notifier:
				for client := range broker.clients {
					client <- event
				}
			case <-broker.quitc:
				return
			}
		}
	}()

	// setup PubSub monitor
	go func() {
		redisCli := redis.NewClient(&redis.Options{
			Addr: "localhost:6379",
		})

		if err := redisCli.Ping().Err(); err != nil {
			// log
			return
		}

		psCli := redisCli.PSubscribe("foo")
		defer psCli.Close()

		for {
			value, err := psCli.Receive()
			if err != nil {
				// log
			}
			fmt.Println("Received...")

			if msg, _ := value.(*redis.Message); msg != nil {
				broker.Notifier <- []byte(msg.Payload)
			}
		}
	}()
	return http.ListenAndServe("localhost:8000", broker)
}

// ServeHTTP wraps HTTP handlers
func (broker *Broker) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	// support flusing
	flusher, ok := rw.(http.Flusher)
	if !ok {
		http.Error(rw, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	// set basic header to support keep-alive HTTP connections
	// this related to event streaming
	rw.Header().Set("Content-Type", "text/event-stream")
	rw.Header().Set("Cache-Control", "no-cache")
	rw.Header().Set("Connection", "keep-alive")
	rw.Header().Set("Access-Control-Allow-Origin", "*")

	// register its own message channel
	messageChan := make(chan MessageChan)

	broker.newClients <- messageChan

	req.ParseForm()
	raw := len(req.Form["raw"]) > 0

	// make sure remove from server
	defer func() {
		broker.closingClients <- messageChan
	}()

	// listen to connection close
	notify := rw.(http.CloseNotifier).CloseNotify()

	go func() {
		<-notify
		broker.closingClients <- messageChan
	}()

	// waiting for messages broadcast
	for {
		if raw {
			fmt.Fprintf(rw, "%s\n", <-messageChan)
		} else {
			fmt.Fprintf(rw, "data: %s\n\n", <-messageChan)
		}

		// flush the data
		flusher.Flush()
	}
}

// Close -
func (broker *Broker) Close() {
	close(broker.quitc)
}
