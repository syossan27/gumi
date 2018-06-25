package main

import (
	"sync"
	"net/http"
	"strconv"
	"fmt"
	"context"
	"io/ioutil"
	"net"
	"time"
	"golang.org/x/sync/semaphore"
	"golang.org/x/time/rate"
)

const(
	Limit = 1024
	Weight = 1
	Burst = 600
)

var (
	client *http.Client
	limit = rate.Every(time.Second / Burst)
	limitter = rate.NewLimiter(limit, Burst)
)

func main() {
	host := "icecrusher.dev-ice01.istyle.local"
	ip, err := net.LookupIP(host)
	if err != nil {
		fmt.Println("Failed resolve")
		return
	}

	defaultRoundTripper := http.DefaultTransport
	defaultTransportPointer, ok := defaultRoundTripper.(*http.Transport)
	if !ok {
		panic("hoge")
	}
	defaultTransport := *defaultTransportPointer
	defaultTransport.MaxIdleConns = 0
	defaultTransport.MaxIdleConnsPerHost = 10000

	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
		DualStack: true,
	}
	defaultTransport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		if addr == host+":80" {
			addr = ip[0].String() + ":80"
		}
		return dialer.DialContext(ctx, network, addr)
	}

	client = &http.Client{Transport: &defaultTransport}

	var wg = sync.WaitGroup{}
	s := semaphore.NewWeighted(Limit)
	for i := 0; i < 10000000; i++ {
		wg.Add(1)
		s.Acquire(context.Background(), Weight)
		go func(id int) {
			if err := limitter.Wait(context.Background()); err != nil {
				fmt.Println("error!")
				return
			}

			defer func() {
				s.Release(Weight)
				wg.Done()
			}()
			idStr := strconv.Itoa(id)
			resp, err := client.Get("http://" + host + "?productId=" + idStr)
			// resp, err := client.Get("http://" + host)
			if err != nil {
				fmt.Println("Error: " + err.Error())
				return
			}
			defer resp.Body.Close()
			_, err = ioutil.ReadAll(resp.Body)
			if err != nil {
				fmt.Println(err.Error())
				return
			}
			fmt.Println("Success: " + idStr)
		}(i)
	}
	wg.Wait()
}
