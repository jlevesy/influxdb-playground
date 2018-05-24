package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"gopkg.in/alexcesaro/statsd.v2"
)

var (
	randLock       sync.Mutex
	zipf           = rand.NewZipf(rand.New(rand.NewSource(0)), 1.1, 1, 1000)
	statsdClient   *statsd.Client
	slownessFixed  = false
	minError       = 95
	minClientError = 85
)

func init() {
	var err error
	statsdClient, err = statsd.New(statsd.Address("telegraf:8125"))

	if err != nil {
		panic(err)
	}

	if os.Getenv("SLOWNESS_FIXED") == "true" {
		slownessFixed = true
	}

	if val, err := strconv.ParseInt(os.Getenv("MIN_ERROR"), 10, 64); err == nil {
		minError = int(val)
	}

	if val, err := strconv.ParseInt(os.Getenv("MIN_CLIENT_ERROR"), 10, 64); err == nil {
		minClientError = int(val)
	}

	fmt.Printf(
		"Starting backend with: slownessFixed %v, mixError %d, minClientError %d \n",
		slownessFixed,
		minError,
		minClientError,
	)
}

func handleHi(w http.ResponseWriter, r *http.Request) {
	t := statsdClient.NewTiming()
	defer func() {
		t.Send("backend.hi.duration")
	}()

	statsdClient.Increment("backend.hi.total")
	var queryTime time.Duration

	if slownessFixed {
		queryTime = 200*time.Millisecond + time.Duration(rand.Intn(100))*time.Millisecond
	} else {
		randLock.Lock()
		queryTime = time.Duration(zipf.Uint64()) * time.Millisecond
		randLock.Unlock()
	}

	time.Sleep(queryTime)

	// Fail sometimes.
	switch v := rand.Intn(100); {
	case v > minError:
		statsdClient.Increment("backend.hi.500")
		w.WriteHeader(500)
		return
	case v > minClientError:
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
