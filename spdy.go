package main

// import (
// 	"github.com/direvius/spdy"
// )

// type SpdyUser struct {
// 	name      string               // user identifier
// 	target    string               // target host and port
// 	batchSize int                  // size of one batch
// 	pacing    float64              // pauses between batches
// 	requests  <-chan *http.Request // source of requests
// 	results   chan<- *PhoutSample  // result sink
// 	client    *spdy.Client         // SPDY client connection
// }

// func (u *User) shoot(req *http.Request, tag string) {

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
