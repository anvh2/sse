package sse

import (
	"fmt"
	"log"
	"net/http"
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
	borker := &Broker{
		Notifier:       make(chan MessageChan, 1),
		newClients:     make(chan chan MessageChan),
		closingClients: make(chan chan MessageChan),
		clients:        make(map[chan MessageChan]bool),
		quitc:          make(chan struct{}),
	}

	go borker.run()

	return borker
}

func (broker *Broker) run() {
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
