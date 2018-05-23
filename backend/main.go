package main

import (
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"gopkg.in/alexcesaro/statsd.v2"
)

var (
	randLock     sync.Mutex
	zipf         = rand.NewZipf(rand.New(rand.NewSource(0)), 1.1, 1, 1000)
	statsdClient *statsd.Client
)

func init() {
	var err error
	statsdClient, err = statsd.New(statsd.Address("telegraf:8125"))

	if err != nil {
		panic(err)
	}
}

func handleHi(w http.ResponseWriter, r *http.Request) {
	t := statsdClient.NewTiming()
	defer func() {
		t.Send("backend.hi.duration")
	}()

	statsdClient.Increment("backend.hi.total")
	randLock.Lock()
	time.Sleep(time.Duration(zipf.Uint64()) * time.Millisecond)
	randLock.Unlock()

	// Fail sometimes.
	switch v := rand.Intn(100); {
	case v > 95:
		statsdClient.Increment("backend.hi.500")
		w.WriteHeader(500)
		return
	case v > 85:
		statsdClient.Increment("backend.hi.400")
		w.WriteHeader(400)
		return
	}

	statsdClient.Increment("backend.hi.200")
	// Return page content.
	w.Write([]byte("hi\n"))
}

func main() {
	defer statsdClient.Close()
	http.HandleFunc("/hi", handleHi)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
