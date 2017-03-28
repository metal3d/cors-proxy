package main

// cors-proxy is a reverse proxy to allow cross origin
// requests (eg. from a javascript XHTTPRequest) to
// another service that doesn't respond to OPTIONS requests.
//
// Author: Patrice FERLET <metal3d@gmail.com>
// Licence: BSD

import (
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
)

var (
	// host:port to connect to.
	portTo = "127.0.0.1:8000"
	// host:port to listen.
	listen = "0.0.0.0:3000"
	// verbose message.
	verbose = false
)

// handleReverseRequest writes back the server response to client.
// If an "OPTIONS" request is called, we
// only return Access-Control-Allow-* to let XHttpRequest working.
func handleReverseRequest(w http.ResponseWriter, r *http.Request) {

	// check scheme
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}

	// build url
	toCall := fmt.Sprintf("%s://%s%s", scheme, portTo, r.URL.String())
	debug("Create request for ", toCall)

	// always allow access origin
	w.Header().Add("Access-Control-Allow-Origin", "*")
	w.Header().Add("Access-Control-Allow-Methods", "GET, PUT, POST, HEAD, TRACE, DELETE, PATCH, COPY, HEAD, LINK, OPTIONS")

	if r.Method == "OPTIONS" {
		debug("CORS asked for ", toCall)
		for n, h := range r.Header {
			if strings.Contains(n, "Access-Control-Request") {
				for _, h := range h {
					k := strings.Replace(n, "Request", "Allow", 1)
					w.Header().Add(k, h)
				}
			}
		}
		// end
		return
	}

	// create request to server
	req, err := http.NewRequest(r.Method, toCall, r.Body)

	// add ALL header to the connection
	for n, h := range r.Header {
		for _, h := range h {
			req.Header.Add(n, h)
		}
	}

	// create a basic client to send request
	client := http.Client{}
	if r.TLS != nil {
		client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	for h, v := range resp.Header {
		for _, v := range v {
			w.Header().Add(h, v)
		}
	}
	// copy the reponse from server to the connected client request
	w.WriteHeader(resp.StatusCode)

	wr, err := io.Copy(w, resp.Body)
	if err != nil {
		log.Println(wr, err)
	} else {
		debug("Writen", wr, "bytes")
	}

}

// validateFlags checks if host:port format is ok.
func validateFlags() {
	for _, f := range []string{portTo, listen} {
		if !strings.Contains(f, ":") {
			log.Fatalf("%s is not right, you must use a coma mark to separate host and port", f)
		}

	}

	parts := strings.Split(portTo, ":")
	if parts[0] == "" {
		log.Println("You didn't set host to connect, using 127.0.0.1:" + parts[1])
		portTo = "127.0.0.1:" + parts[1]
	}

}

// debug writes message when verbose flag is true.
func debug(v ...interface{}) {
	if verbose {
		log.Println(v...)
	}
}

func main() {
	flag.StringVar(&portTo, "p", portTo, "service port")
	flag.StringVar(&listen, "l", listen, "listen interface")
	flag.BoolVar(&verbose, "v", verbose, "verbose")
	flag.Parse()

	validateFlags()
	http.HandleFunc("/", handleReverseRequest)
	log.Println(listen, "-->", portTo)
	http.ListenAndServe(listen, nil)
}
