package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/roman-mazur/design-practice-3-template/httptools"
	"github.com/roman-mazur/design-practice-3-template/signal"
)

var (
	port       = flag.Int("port", 8090, "load balancer port")
	timeoutSec = flag.Int("timeout-sec", 3, "request timeout time in seconds")
	https      = flag.Bool("https", false, "whether backends support HTTPs")

	traceEnabled = flag.Bool("trace", false, "whether to include tracing information into responses")
)

var (
	timeout     = time.Duration(*timeoutSec) * time.Second
	serversPool = []string{
		"server1:8080",
		"server2:8080",
		"server3:8080",
	}
)

func scheme() string {
	if *https {
		return "https"
	}
	return "http"
}

func health(dst string) bool {
	ctx, _ := context.WithTimeout(context.Background(), timeout)
	req, _ := http.NewRequestWithContext(ctx, "GET",
		fmt.Sprintf("%s://%s/health", scheme(), dst), nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	if resp.StatusCode != http.StatusOK {
		return false
	}
	return true
}

func forward(dst string, rw http.ResponseWriter, r *http.Request) error {
	ctx, _ := context.WithTimeout(r.Context(), timeout)
	fwdRequest := r.Clone(ctx)
	fwdRequest.RequestURI = ""
	fwdRequest.URL.Host = dst
	fwdRequest.URL.Scheme = scheme()
	fwdRequest.Host = dst

	resp, err := http.DefaultClient.Do(fwdRequest)
	if err == nil {
		for k, values := range resp.Header {
			for _, value := range values {
				rw.Header().Add(k, value)
			}
		}
		if *traceEnabled {
			rw.Header().Set("lb-from", dst)
		}
		log.Println("fwd", resp.StatusCode, resp.Request.URL)
		rw.WriteHeader(resp.StatusCode)
		defer resp.Body.Close()
		_, err := io.Copy(rw, resp.Body)
		if err != nil {
			log.Printf("Failed to write response: %s", err)
		}
		return nil
	} else {
		log.Printf("Failed to get response from %s: %s", dst, err)
		rw.WriteHeader(http.StatusServiceUnavailable)
		return err
	}
}
func findServer(index int, availableAnyServer bool) (int, error, bool) {

	minTraffic := trafficOfServer(serversPool[0])
	for i := 1; i < len(serversPool); i++ {
		currTraffic := trafficOfServer(serversPool[i])
		if health(serversPool[i]) && currTraffic <= minTraffic {
			index = i
			minTraffic = currTraffic
			availableAnyServer = true
		}
	}
	if availableAnyServer == false {
		return 0, errors.New("No servers available"), false
	}
	return index, nil, true
}

func trafficOfServer(dst string) int {
	ctx, _ := context.WithTimeout(context.Background(), timeout)
	req, _ := http.NewRequestWithContext(ctx, "GET",
		fmt.Sprintf("%s://%s/trafficOfServer", scheme(), dst), nil)
	resp, _ := http.DefaultClient.Do(req)
	respBuf := new(bytes.Buffer)
	respBuf.ReadFrom(resp.Body)
	respString := string(respBuf.Bytes())
	respInt, _ := strconv.Atoi(respString)
	return respInt

}

func main() {
	flag.Parse()
	availableAnyServer := false
	var err error = nil
	index := 0
	// TODO: Використовуйте дані про стан сервреа, щоб підтримувати список тих серверів, яким можна відправляти ззапит.
	for _, server := range serversPool {
		server := server
		go func() {
			for range time.Tick(10 * time.Second) {
				log.Println(server, health(server))
			}
		}()
	}

	frontend := httptools.CreateServer(*port, http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		// TODO: Рееалізуйте свій алгоритм балансувальника.
		index, err, availableAnyServer = findServer(index, availableAnyServer)
		if err != nil && availableAnyServer == false {
			log.Printf("Errors: %s", err.Error())
			rw.WriteHeader(http.StatusInternalServerError)
			rw.Write([]byte(err.Error()))
		} else {
			forward(serversPool[index], rw, r)
		}

		availableAnyServer = false
		err = nil
	}))

	log.Println("Starting load balancer...")
	log.Printf("Tracing support enabled: %t", *traceEnabled)
	frontend.Start()
	signal.WaitForTerminationSignal()
}

// function for testing
func testBalancer(testTraffic [3]int, testHealth [3]bool) (int, error, bool) {
	availableAnyServerTest := false
	index := 0
	minTraffic := 2000
	for i := 0; i < len(testTraffic); i++ {
		if testHealth[i] && (testTraffic[i] <= minTraffic) {
			index = i
			minTraffic = testTraffic[i]
			availableAnyServerTest = true
		}
	}
	if !availableAnyServerTest {
		return 0, errors.New("No servers available"), false
	}
	return index, nil, true
}
