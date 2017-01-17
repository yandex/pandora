// Copyright 2014 Jamie Hall. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package spdy_test

import (
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/SlyMarbo/spdy"
)

func init() {
	// spdy.EnableDebugOutput()
}

func TestClient(t *testing.T) {
	ts := newServer(robotsTxtHandler)
	defer ts.Close()

	client := newClient()

	r, err := client.Get(ts.URL)
	var b []byte
	if err == nil {
		b, err = pedanticReadAll(r.Body)
		r.Body.Close()
	}
	if err != nil {
		t.Error(err)
	} else if s := string(b); !strings.HasPrefix(s, "User-agent:") {
		t.Errorf("Incorrect page body (did not begin with User-agent): %q", s)
	}
}

func TestClientHead(t *testing.T) {
	ts := newServer(robotsTxtHandler)
	defer ts.Close()

	client := newClient()
	r, err := client.Head(ts.URL)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := r.Header["Last-Modified"]; !ok {
		t.Error("Last-Modified header not found.")
	}
}

func TestClientInGoroutines(t *testing.T) {
	ts := newServer(robotsTxtHandler)
	ts.Config.ErrorLog = log.New(ioutil.Discard, "", 0) // ignore messages
	defer ts.Close()

	client := spdy.NewClient(false)

	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var b []byte
			r, err := client.Get(ts.URL)
			if err == nil {
				b, err = pedanticReadAll(r.Body)
				r.Body.Close()
			}
			if err != nil {
				// We've turned off InsecureSkipVerify, so
				// ignore bad cert warnings.
				if !strings.Contains(err.Error(), "certificate signed by unknown authority") {
					t.Error(err)
				}
			} else if s := string(b); !strings.HasPrefix(s, "User-agent:") {
				t.Errorf("Incorrect page body (did not begin with User-agent): %q", s)
			}
		}()
	}

	wg.Wait()
}

// FIXME: Fails
// func TestClientRedirects(t *testing.T) {
// 	defer afterTest(t)
// 	var ts *httptest.Server
// 	ts = newServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		n, _ := strconv.Atoi(r.FormValue("n"))
// 		// Test Referer header. (7 is arbitrary position to test at)
// 		if n == 7 {
// 			if g, e := r.Referer(), ts.URL+"/?n=6"; e != g {
// 				t.Errorf("on request ?n=7, expected referer of %q; got %q", e, g)
// 			}
// 		}
// 		if n < 15 {
// 			http.Redirect(w, r, fmt.Sprintf("/?n=%d", n+1), http.StatusFound)
// 			return
// 		}
// 		fmt.Fprintf(w, "n=%d", n)
// 	}))
// 	defer ts.Close()

// 	c := newClient()
// 	_, err := c.Get(ts.URL)
// 	if e, g := "Get /?n=10: stopped after 10 redirects", fmt.Sprintf("%v", err); e != g {
// 		t.Errorf("with default client Get, expected error %q, got %q", e, g)
// 	}

// 	// HEAD request should also have the ability to follow redirects.
// 	_, err = c.Head(ts.URL)
// 	if e, g := "Head /?n=10: stopped after 10 redirects", fmt.Sprintf("%v", err); e != g {
// 		t.Errorf("with default client Head, expected error %q, got %q", e, g)
// 	}

// 	// Do should also follow redirects.
// 	greq, _ := http.NewRequest("GET", ts.URL, nil)
// 	_, err = c.Do(greq)
// 	if e, g := "Get /?n=10: stopped after 10 redirects", fmt.Sprintf("%v", err); e != g {
// 		t.Errorf("with default client Do, expected error %q, got %q", e, g)
// 	}

// 	var checkErr error
// 	var lastVia []*http.Request
// 	c = newClient()
// 	c.CheckRedirect = func(_ *http.Request, via []*http.Request) error {
// 		lastVia = via
// 		return checkErr
// 	}
// 	res, err := c.Get(ts.URL)
// 	if err != nil {
// 		t.Fatalf("Get error: %v", err)
// 	}
// 	res.Body.Close()
// 	finalUrl := res.Request.URL.String()
// 	if e, g := "<nil>", fmt.Sprintf("%v", err); e != g {
// 		t.Errorf("with custom client, expected error %q, got %q", e, g)
// 	}
// 	if !strings.HasSuffix(finalUrl, "/?n=15") {
// 		t.Errorf("expected final url to end in /?n=15; got url %q", finalUrl)
// 	}
// 	if e, g := 15, len(lastVia); e != g {
// 		t.Errorf("expected lastVia to have contained %d elements; got %d", e, g)
// 	}

// 	checkErr = errors.New("no redirects allowed")
// 	res, err = c.Get(ts.URL)
// 	if urlError, ok := err.(*url.Error); !ok || urlError.Err != checkErr {
// 		t.Errorf("with redirects forbidden, expected a *url.Error with our 'no redirects allowed' error inside; got %#v (%q)", err, err)
// 	}
// 	if res == nil {
// 		t.Fatalf("Expected a non-nil Response on CheckRedirect failure (http://golang.org/issue/3795)")
// 	}
// 	res.Body.Close()
// 	if res.Header.Get("Location") == "" {
// 		t.Errorf("no Location header in Response")
// 	}
// }

// FIXME: Deadlocks
// func TestPostRedirects(t *testing.T) {
//	defer afterTest(t)
// 	var log struct {
// 		sync.Mutex
// 		bytes.Buffer
// 	}
// 	var ts *httptest.Server
// 	ts = newServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		log.Lock()
// 		fmt.Fprintf(&log.Buffer, "%s %s ", r.Method, r.RequestURI)
// 		log.Unlock()
// 		if v := r.URL.Query().Get("code"); v != "" {
// 			code, _ := strconv.Atoi(v)
// 			if code/100 == 3 {
// 				w.Header().Set("Location", ts.URL)
// 			}
// 			w.WriteHeader(code)
// 		}
// 	}))
// 	defer ts.Close()
// 	tests := []struct {
// 		suffix string
// 		want   int // response code
// 	}{
// 		{"/", 200},
// 		{"/?code=301", 301},
// 		{"/?code=302", 200},
// 		{"/?code=303", 200},
// 		{"/?code=404", 404},
// 	}
// 	client := newClient()
// 	for _, tt := range tests {
// 		res, err := client.Post(ts.URL+tt.suffix, "text/plain", strings.NewReader("Some content"))
// 		if err != nil {
// 			t.Fatal(err)
// 		}
// 		if res.StatusCode != tt.want {
// 			t.Errorf("POST %s: status code = %d; want %d", tt.suffix, res.StatusCode, tt.want)
// 		}
// 	}
// 	log.Lock()
// 	got := log.String()
// 	log.Unlock()
// 	want := "POST / POST /?code=301 POST /?code=302 GET / POST /?code=303 GET / POST /?code=404 "
// 	if got != want {
// 		t.Errorf("Log differs.\n Got: %q\nWant: %q", got, want)
// 	}
// }

// FIXME: Fails (not a Flusher)
// func TestStreamingGet(t *testing.T) {
// 	defer afterTest(t)
// 	say := make(chan string)
// 	ts := newServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		w.(http.Flusher).Flush()
// 		for str := range say {
// 			w.Write([]byte(str))
// 			w.(http.Flusher).Flush()
// 		}
// 	}))
// 	defer ts.Close()

// 	c := newClient()
// 	res, err := c.Get(ts.URL + "/")
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	var buf [10]byte
// 	for _, str := range []string{"i", "am", "also", "known", "as", "comet"} {
// 		say <- str
// 		n, err := io.ReadFull(res.Body, buf[0:len(str)])
// 		if err != nil {
// 			t.Fatalf("ReadFull on %q: %v", str, err)
// 		}
// 		if n != len(str) {
// 			t.Fatalf("Receiving %q, only read %d bytes", str, n)
// 		}
// 		got := string(buf[0:n])
// 		if got != str {
// 			t.Fatalf("Expected %q, got %q", str, got)
// 		}
// 	}
// 	close(say)
// 	_, err = io.ReadFull(res.Body, buf[0:1])
// 	if err != io.EOF {
// 		t.Fatalf("at end expected EOF, got %v", err)
// 	}
// }

//
// HELPERS
//

var robotsTxtHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Last-Modified", "sometime")
	fmt.Fprintf(w, "User-agent: go\nDisallow: /something/")
})

// pedanticReadAll works like ioutil.ReadAll but additionally
// verifies that r obeys the documented io.Reader contract.
func pedanticReadAll(r io.Reader) (b []byte, err error) {
	var bufa [64]byte
	buf := bufa[:]
	for {
		n, err := r.Read(buf)
		if n == 0 && err == nil {
			return nil, fmt.Errorf("Read: n=0 with err=nil")
		}
		b = append(b, buf[:n]...)
		if err == io.EOF {
			n, err := r.Read(buf)
			if n != 0 || err != io.EOF {
				return nil, fmt.Errorf("Read: n=%d err=%#v after EOF", n, err)
			}
			return b, nil
		}
		if err != nil {
			return b, err
		}
	}
}

func newServer(handler http.Handler) *httptest.Server {
	ts := httptest.NewUnstartedServer(handler)
	spdy.AddSPDY(ts.Config)
	ts.TLS = ts.Config.TLSConfig
	ts.StartTLS()
	return ts
}

func newClient() *http.Client {
	tr := spdy.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
			NextProtos:         []string{"spdy/3.1"},
		},
	}
	return &http.Client{Transport: &tr}
}
