package status

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"math/rand"
	"monitor/log"
	"net"
	"net/http"
	"os"
	"time"
)

func PushFile(jsonData []byte, filePath string) {
	WORKER_HOST := os.Getenv("WORKER_HOST")
	if WORKER_HOST == "" {
		panic("WORKER_HOST is not set")
	}
	WORKER_PORT := os.Getenv("WORKER_PORT")
	if WORKER_PORT == "" {
		panic("WORKER_PORT is not set")
	}

	workerHosts := GetWorkerHosts(WORKER_HOST)

	client := &http.Client{}
	basic_auth_user := os.Getenv("WORKER_BASIC_AUTH_USER")
	basic_auth_pass := os.Getenv("WORKER_BASIC_AUTH_PASS")

	for _, workerHost := range workerHosts {
		retry := 2
		for retry > 0 {
			body := &bytes.Buffer{}
			length, writeErr := body.Write(jsonData)
			if writeErr != nil {
				panic("could not write file to buffer: " + writeErr.Error())
			}

			url := "http://" + workerHost + ":" + WORKER_PORT + "/upload/" + filePath
			req, err := http.NewRequest("PUT", url, body)
			if err != nil {
				panic("could not create request: " + err.Error())
			}
			req.Header.Set("Content-Type", "application/binary")
			req.Header.Set("Content-Length", fmt.Sprint(length))
			basic_auth := base64.StdEncoding.EncodeToString([]byte(basic_auth_user + ":" + basic_auth_pass))
			req.Header.Set("Authorization", "Basic "+basic_auth)
			resp, err := client.Do(req)
			if err != nil {
				panic("could not send request: " + err.Error())
			}
			statusOK := resp.StatusCode >= 200 && resp.StatusCode < 300
			if statusOK {
				// All good
				// log.Info.Println(filePath + " pushed successfully")
				break
			}

			log.Error.Println("response status: " + resp.Status)
			resBody, err := io.ReadAll(resp.Body)
			if err != nil {
				panic("client: could not read response body: " + err.Error())
			}
			log.Error.Printf("client: response body: %s\n", resBody)
			log.Error.Println("could not push " + filePath + ", retrying...")
			retry--
			// Wait random time between 1 and 5 seconds
			waitingTime := time.Duration(1 + 4*rand.Float64())
			time.Sleep(waitingTime)
		}
	}
}

func GetWorkerHosts(workerHost string) []string {
	workerHosts, err := net.LookupHost(workerHost)
	if err != nil {
		panic("could not resolve WORKER_HOST, error: " + err.Error())
	}

	return workerHosts
}
