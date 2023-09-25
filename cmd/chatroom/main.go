package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/dustin/go-broadcast"
)

var chatChannel broadcast.Broadcaster = nil

func main() {
	chatChannel = broadcast.NewBroadcaster(100)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "web/index.html")
	})
	http.HandleFunc("/chat", chatHandler)
	http.HandleFunc("/htmx/chat", chatHandlerHtmx)

	fmt.Println("starting server")
	log.Fatal(http.ListenAndServe("0.0.0.0:8081", nil))
	chatChannel.Close()
}

func chatHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		chatListener(w, r)
	} else if r.Method == "POST" {
		chat(w, r)
	}
}

func chat(w http.ResponseWriter, r *http.Request) {
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		fmt.Printf("could not read body: %s\n", err)
		w.WriteHeader(400)
		return
	}

	chat := string(bodyBytes)

	fmt.Println("sent: ", chat)
	chatChannel.Submit(chat)
}

func chatListener(w http.ResponseWriter, r *http.Request) {
	ch := make(chan interface{})
	chatChannel.Register(ch)
	defer chatChannel.Unregister(ch)
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Content-Type", "text/event-stream")

	w.(http.Flusher).Flush()
	fmt.Print("Listening\n")
loop:
	for {
		var chat interface{}
		select {
		case chat = <-ch:
			fmt.Fprintf(w, "data: %s \n\n", chat)
			w.(http.Flusher).Flush()
		case <-r.Context().Done():
			break loop
		}
	}
	fmt.Print("done Listening\n")
}

func chatHandlerHtmx(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		chatListenerHtmx(w, r)
	} else if r.Method == "POST" {
		chatHtmx(w, r)
	}
}

func chatHtmx(w http.ResponseWriter, r *http.Request) {
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		fmt.Printf("could not read body: %s\n", err)
		w.WriteHeader(400)
		return
	}

	// body comes in as an array of bytes that does url encoding, so we must convert it to a string and the unencode it
	encodedBody := string(bodyBytes)
	body, err := url.QueryUnescape(encodedBody)
	if err != nil {
		fmt.Printf("could unencode body: %s\n", err)
		w.WriteHeader(400)
		return
	}

	var chat string

	if strings.HasPrefix(body, "chat") {
		chat = strings.Split(body, "=")[1]
	} else {
		fmt.Printf("parsing error\n")
		w.WriteHeader(400)
		return
	}

	fmt.Println("sent: ", chat)
	chatChannel.Submit(chat)
}

func chatListenerHtmx(w http.ResponseWriter, r *http.Request) {
	ch := make(chan interface{})
	chatChannel.Register(ch)
	defer chatChannel.Unregister(ch)
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Content-Type", "text/event-stream")

	w.(http.Flusher).Flush()
	fmt.Print("Listening\n")
loop:
	for {
		var chat interface{}
		select {
		case chat = <-ch:
			fmt.Fprintf(w, "data: <p>%s</p> \nevent: chat\n\n", chat)
			w.(http.Flusher).Flush()
		case <-r.Context().Done():
			break loop
		}
	}
	fmt.Print("done Listening\n")
}
