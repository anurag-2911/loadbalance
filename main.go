package main

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync/atomic"
)

func main() {
	fmt.Println("load balancer running")
	http.HandleFunc("/", roundRobinBalance)
	http.ListenAndServe(":8080", nil)
}

var servers = []string{
	"http://localhost:8081",
	"http://localhost:8082",
	"http://localhost:8083",
}

var counter int32

func roundRobinBalance(w http.ResponseWriter, r *http.Request) {

	fmt.Println("request came ", r.URL.Path)
	serverIndex := atomic.AddInt32(&counter, 1)

	server := servers[serverIndex%int32(len(servers))]

	target, err := url.Parse(server)
	if err != nil {
		fmt.Printf("error in parsing the url string %s", err)
		return
	}
	proxy := httputil.NewSingleHostReverseProxy(target)

	proxy.ServeHTTP(w, r)

	fmt.Println("request served ", r.URL.Path)
}
