package main

import (
	"flag"
	"fmt"
	"github.com/ruudk/dead-code-analyzer/server/collector"
	"github.com/ruudk/dead-code-analyzer/server/web"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var httpPort int
var collectorPort int
var storageFile string
var verbose bool
var ServiceVersion = "dev"

func main() {
	flag.IntVar(&httpPort, "httpPort", 8080, "httpPort")
	flag.IntVar(&collectorPort, "collectorPort", 8125, "collectorPort")
	flag.StringVar(&storageFile, "storageFile", "data.json", "path to store the data")
	flag.BoolVar(&verbose, "verbose", false, "verbosity")
	flag.Parse()

	log.Printf("Starting Dead Code Detector (%s)", ServiceVersion)

	col, err := collector.NewCollector(storageFile)
	if err != nil {
		log.Fatalf("cannot start collector: %s", err)
	}

	col.Mutex.RLock()
	fmt.Printf("Total length of Autoloaded classes: %d\n", len(col.Storage.AutoLoaded))
	var dead = 0
	var active = 0
	for _, i := range col.Storage.AutoLoaded {
		if i == 0 {
			dead++
		} else {
			active++
		}
	}
	col.Mutex.RUnlock()

	fmt.Printf("Dead %d Active %d\n", dead, active)

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Println("Saving...")
		col.Save()
		os.Exit(1)
	}()

	go func() {
		for {
			time.Sleep(10 * time.Second)
			col.Save()
		}
	}()

	w := web.NewWebServer(col, httpPort)

	go col.Listen(collectorPort)

	log.Fatal(w.ListenAndServe())
}
