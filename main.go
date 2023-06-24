package main

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"sync/atomic"
	"time"
)

type ServerStatus struct {
	Url   *url.URL
	Alive bool
	Mutex sync.RWMutex
}

func main() {
	fmt.Println("load balancer running")
	done := make(chan struct{})
	for _, s := range servers {
		go healthCheckWorker(s, done, 10*time.Second)
	}
	http.HandleFunc("/", roundRobinBalance)
	http.ListenAndServe(":8080", nil)
	close(done)
}

var servers = []*ServerStatus{
	{Url: &url.URL{Scheme: "http", Host: "localhost:8081"}},
	{Url: &url.URL{Scheme: "http", Host: "localhost:8082"}},
	{Url: &url.URL{Scheme: "http", Host: "localhost:8083"}},
}

var counter int32

func roundRobinBalance(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("round robin balancer %s\n", r.URL.RawPath)
	var server *ServerStatus
	for i := 0; i < len(servers); i++ {
		serverIndex := atomic.AddInt32(&counter, 1)
		server = servers[serverIndex%int32(len(servers))]

		server.Mutex.Lock()
		alive := server.Alive
		fmt.Printf("server : %s, alive %v\n", server.Url, server.Alive)
		server.Mutex.Unlock()
		if alive {
			break
		}
		server = nil
	}
	if server == nil {
		http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
		return
	}
	fmt.Println("serving by ", server.Url)
	proxy := httputil.NewSingleHostReverseProxy(server.Url)
	proxy.ServeHTTP(w, r)

}

func (server *ServerStatus) HealthCheck() {
	fmt.Printf("health check for url %s\n",server.Url.String())
	resp, err := http.Get(server.Url.String())
	if err != nil {
		fmt.Printf("health check failed for %s \n", server.Url.String())
		server.Mutex.Lock()
		server.Alive = false
		server.Mutex.Unlock()
		return
	}
	defer resp.Body.Close()
	server.Mutex.Lock()
	fmt.Printf("status code %v for server %s\n", resp.Status, server.Url.String())
	server.Alive = resp.StatusCode == http.StatusOK
	server.Mutex.Unlock()
}
func healthCheckWorker(server *ServerStatus, done <-chan struct{}, interval time.Duration) {
	ticker := time.NewTicker(interval)
	for {
		select {
		case <-done:
			ticker.Stop()
			fmt.Printf("stopping health check %v\n", server.Url.String())
			return
		case <-ticker.C:
			fmt.Printf("health check for %s\n", server.Url.String())
			server.HealthCheck()
		}

	}
}
