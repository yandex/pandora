package main

import (
	"log"
)

// import (
// 	"bufio"
// 	"crypto/tls"
// 	"encoding/json"
// 	"flag"
// 	"fmt"
// 	"github.com/direvius/spdy"
// 	"github.com/nu7hatch/gouuid"
// 	"io"
// 	"io/ioutil"
// 	"log"
// 	"net/http"
// 	"os"
// 	"time"
// )

// // TODO: loop ammo
// // TODO: too verbose tags. Lunapark is very sad about that
// // TODO: constructor for PhoutSample
// // TODO: exit not after 100 seconds, but when all is finished

// type User struct {
// 	name      string               // user identifier
// 	target    string               // target host and port
// 	batchSize int                  // size of one batch
// 	pacing    float64              // pauses between batches
// 	requests  <-chan *http.Request // source of requests
// 	client    *spdy.Client         // SPDY client connection
// }

// type Ammo struct {
// 	Host    string
// 	Method  string
// 	Uri     string
// 	Headers map[string]string
// }

type PhoutSample struct {
	ts             float64
	tag            string
	rt             int
	connect        int
	send           int
	latency        int
	receive        int
	interval_event int
	egress         int
	igress         int
	netCode        int
	protoCode      int
}

func main() {
	log.Println("Done")
}

// func (ps *PhoutSample) String() string {
// 	return fmt.Sprintf(
// 		"%.3f\t%s\t%d\t"+
// 			"%d\t%d\t"+
// 			"%d\t%d\t"+
// 			"%d\t"+
// 			"%d\t%d\t"+
// 			"%d\t%d",
// 		ps.ts, ps.tag, ps.rt,
// 		ps.connect, ps.send,
// 		ps.latency, ps.receive,
// 		ps.interval_event,
// 		ps.egress, ps.igress,
// 		ps.netCode, ps.protoCode,
// 	)
// }

// func makeRequest(a Ammo) (req *http.Request, err error) {
// 	//make a request
// 	req, err = http.NewRequest(a.Method, "https://"+a.Host+a.Uri, nil)
// 	for k, v := range a.Headers {
// 		req.Header.Set(k, v)
// 	}
// 	return
// }

// func getBatch(src <-chan *http.Request, size int) []*http.Request {
// 	res := make([]*http.Request, size)
// 	for i := 0; i < size; i++ {
// 		res[i] = <-src
// 	}
// 	return res
// }

// func (u *User) shoot(batch []*http.Request) {

// 	batchUUID, _ := uuid.NewV4()

// 	for i, req := range batch {
// 		if u.client == nil {
// 			log.Printf("Connecting %s...\n", u.name)
// 			connectStart := time.Now()
// 			ps := PhoutSample{ts: float64(connectStart.UnixNano()) / 1e9, tag: fmt.Sprintf(
// 				"%s|%v|%d", u.name, "CONNECT", i)}
// 			if err := u.connect(); err != nil {
// 				log.Printf("Failed to connect %s: %s\n", u.name, err)
// 				ps.netCode = 999
// 				ps.protoCode = 500
// 				ps.rt = int(time.Since(connectStart).Seconds() * 1e6)
// 				results <- &ps
// 				return
// 			} else {

// 				ps.protoCode = 200

// 				ps.rt = int(time.Since(connectStart).Seconds() * 1e6)
// 				results <- &ps
// 			}
// 		}
// 		start := time.Now()
// 		ps := PhoutSample{ts: float64(start.UnixNano()) / 1e9, tag: fmt.Sprintf(
// 			"%s|%v|%d", u.name, batchUUID, i)}
// 		// now send the request to obtain a http response
// 		res, err := u.client.Do(req)
// 		if err != nil {
// 			fmt.Printf("client: request: %s\n", err)
// 			ps.netCode = 999
// 			ps.protoCode = 500
// 		} else {
// 			// now handle the response
// 			//data := make([]byte, int(res.ContentLength))
// 			_, err = io.Copy(ioutil.Discard, res.Body)
// 			// _, err = res.Body.(io.Reader).Read(data)
// 			// fmt.Println(string(data))
// 			res.Body.Close()
// 			ps.protoCode = res.StatusCode
// 		}
// 		ps.rt = int(time.Since(start).Seconds() * 1e6)
// 		results <- &ps
// 	}
// }

// func (u *User) run() {
// 	log.Println("Started", u.name)

// 	//u.connect()

// 	defer u.disconnect()
// 	for {
// 		batch := getBatch(u.requests, u.batchSize)
// 		u.shoot(batch)
// 		time.Sleep(time.Duration(u.pacing) * time.Second)
// 	}
// }

// func (u *User) connect() (err error) {
// 	if u.client != nil {
// 		u.client.Close()
// 	}
// 	config := tls.Config{
// 		InsecureSkipVerify: true,
// 		NextProtos:         []string{"spdy/3.1"},
// 	}

// 	conn, err := tls.Dial("tcp", u.target, &config)
// 	if err != nil {
// 		fmt.Printf("client: dial: %s\n", err)
// 		return
// 	}
// 	u.client, err = spdy.NewClientConn(conn)
// 	if err != nil {
// 		fmt.Printf("client: connect: %s\n", err)
// 		return
// 	}
// 	return
// }

// func (u *User) disconnect() {
// 	if u.client != nil {
// 		u.client.Close()
// 		u.client = nil
// 	}
// }

// var requests = make(chan *http.Request, 100)
// var results = make(chan *PhoutSample)

// func main() {
// 	batchSize := flag.Int("batch", 10, "batch size")
// 	pacing := flag.Float64("pacing", 3.0, "pause between batches")
// 	nclients := flag.Int("users", 100, "users count")
// 	delay := flag.Float64("delay", 0.1, "delay between user starts")
// 	ammoFile := flag.String("ammo", "./ammo.jsonline", "ammo file")
// 	phoutFile := flag.String("phout", "./phout.log", "output data in Phantom's phout format")
// 	target := flag.String("target", "localhost:443", "target host and port")
// 	debug := flag.Bool("debug", false, "enable debug")
// 	flag.Parse()

// 	if *debug {
// 		spdy.EnableDebug()
// 	}

// 	go func() { // requests reader/generator
// 		ammo, _ := os.Open(*ammoFile)
// 		defer ammo.Close()
// 		scanner := bufio.NewScanner(ammo)
// 		scanner.Split(bufio.ScanLines)

// 		for scanner.Scan() {
// 			txt := scanner.Text()
// 			var a Ammo
// 			if err := json.Unmarshal([]byte(txt), &a); err != nil {
// 				log.Println("client: unmarshal: %s", err)
// 			} else {
// 				r, err := makeRequest(a)
// 				if err != nil {
// 					log.Println("client: make request: %s", err)
// 				} else {
// 					requests <- r
// 				}
// 			}
// 		}
// 		close(requests)
// 		log.Println("Ran out of ammo")
// 	}()
// 	go func() { // aggregator
// 		phout, _ := os.Create(*phoutFile)
// 		defer phout.Close()
// 		counter := 0
// 		lastValue := 0
// 		ticker := time.NewTicker(time.Millisecond * 1000)
// 		go func() {
// 			for range ticker.C {
// 				cnt := counter
// 				log.Println("RPS:", cnt-lastValue)
// 				lastValue = cnt
// 			}
// 		}()
// 		for r := range results {
// 			counter++
// 			//fmt.Println("Got sample: ", r)
// 			phout.WriteString(fmt.Sprintf("%v\n", r))
// 		}
// 	}()
// 	for i := 0; i < *nclients; i++ {
// 		time.Sleep(time.Duration(*delay*1e3) * time.Millisecond)
// 		u := User{fmt.Sprintf("user-%d", i), *target, *batchSize, *pacing, requests, nil}
// 		go u.run()
// 	}
// 	time.Sleep(time.Duration(100) * time.Second)
// }
