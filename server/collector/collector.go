package collector

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Collector struct {
	storageFile string
	Mutex       sync.RWMutex
	Storage     *Storage
}

type Storage struct {
	AutoLoaded map[string]int
	Since      time.Time
}

func (c *Collector) Reset() {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()
	c.Storage.AutoLoaded = map[string]int{}
	c.Storage.Since = time.Now()
}

func (c *Collector) IncrementAutoLoadedClass(class string, count int) {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()

	if _, ok := c.Storage.AutoLoaded[class]; ok {
		c.Storage.AutoLoaded[class] += count

		return
	}

	c.Storage.AutoLoaded[class] = count
}

func (c *Collector) RemoveClass(class string) {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()

	delete(c.Storage.AutoLoaded, class)
}

func NewCollector(storageFile string) (*Collector, error) {
	storage := &Storage{
		AutoLoaded: map[string]int{},
		Since:      time.Now(),
	}

	rawData, err := ioutil.ReadFile(storageFile)
	if err == nil {
		err = json.Unmarshal(rawData, &storage)
		if err != nil {
			return nil, err
		}
	}

	return &Collector{
		storageFile:    storageFile,
		Storage: storage,
	}, nil
}

func (c *Collector) Save() error {
	j, _ := json.Marshal(c.Storage)
	return ioutil.WriteFile(c.storageFile, j, 0644)
}

func (c *Collector) Listen(port int) {
	conn, _ := net.ListenUDP("udp", &net.UDPAddr{IP: []byte{0, 0, 0, 0}, Port: port, Zone: ""})
	defer conn.Close()
	buf := make([]byte, 1024)
	for {
		n, _, _ := conn.ReadFromUDP(buf)
		input := string(buf[0:n])
		metrics, err := parseMetrics(input)
		if err != nil {
			fmt.Printf("could not parse metrics: %s\n", err)
			continue
		}

		for _, metric := range metrics {
			switch metric.name {
			case "autoloaded":
				c.IncrementAutoLoadedClass(
					strings.Replace(metric.tags["class"], `/`, `\`, -1),
					metric.increment,
				)
			}
		}
	}
}

type Metric struct {
	name      string
	tags      map[string]string
	increment int
	command   string
}

func parseMetrics(input string) ([]*Metric, error) {
	metrics := make([]*Metric, 0)
	for _, input := range strings.Split(input, "\n") {
		metric, err := parseMetric(input)
		if err != nil {
			fmt.Printf("skipping metric %s: %s\n", input, err)
			continue
		}

		metrics = append(metrics, metric)
	}

	return metrics, nil
}

func parseMetric(input string) (*Metric, error) {
	colonParts := strings.Split(input, ":")
	if len(colonParts) != 2 {
		return nil, fmt.Errorf("invalid input, colon missing")
	}
	commaParts := strings.Split(colonParts[0], ",")
	name := commaParts[0]
	tags := map[string]string{}
	if len(commaParts) > 1 {
		for _, tag := range commaParts[1:] {
			tagPart := strings.Split(tag, "=")
			tags[tagPart[0]] = tagPart[1]
		}
	}
	pipeParts := strings.Split(colonParts[1], "|")
	if len(pipeParts) != 2 {
		return nil, fmt.Errorf("invalid input, pipe missing")
	}
	increment, err := strconv.Atoi(pipeParts[0])
	if err != nil {
		return nil, fmt.Errorf("cannot convert string to int")
	}
	command := pipeParts[1]

	return &Metric{
		name,
		tags,
		increment,
		command,
	}, nil
}
