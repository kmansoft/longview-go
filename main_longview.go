package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"strings"
	"time"
)

const (
	API_KEY_FILE = "/etc/linode/longview.key"
)

type Response struct {
	Sleep int `json:"sleep"`
}

func main() {
	// Read API key
	apiKey := ""
	apiKeyBytes, err := ioutil.ReadFile(API_KEY_FILE)
	if err == nil {
		apiKey = strings.TrimSpace(string(apiKeyBytes))
	}
	if len(apiKey) == 0 {
		// No api key
		log.Printf("There is no API key, please set in %s", API_KEY_FILE)
		return
	} else if !isApiKeyValid(apiKey) {
		// Api key is set but not valid
		log.Printf("The API key is not valid, please update in %s", API_KEY_FILE)
		return
	}

	// Default sleep time
	sleep := 15

	// HTTP client
	client := &http.Client{Timeout: 5 * time.Second}

	for {

		// Data item
		data := Data{
			Instant:   make(map[string]interface{}),
			Longterm:  make(map[string]interface{}),
			Timestamp: time.Now().Unix(),
		}

		// Fill it in
		GetDataMemory(&data)
		GetDataCPU(&data)

		GetDataSysInfo(&data)

		GetDataNetwork(&data)
		GetDataDisks(&data)

		GetDataProcessesPorts(&data)

		GetDataAppNginx(client, &data)

		// Send to server
		sleepNew, err := sendDataToServer(client, apiKey, &data)
		if err != nil {
			sleep = 15
		} else if sleepNew > 0 {
			sleep = sleepNew
		}

		// Wait / sleep
		fmt.Printf("Sleeping for %d seconds\n", sleep)
		time.Sleep(time.Duration(sleep) * time.Second)
	}
}

func isApiKeyValid(apiKey string) bool {
	return len(apiKey) == 35
}

func sendDataToServer(client *http.Client, apiKey string, data *Data) (int, error) {

	// Add other smaller required fields
	post := PostData{
		Version:   "1.0",
		ApiKey:    apiKey,
		Payload:   make([]*Data, 0),
		Timestamp: time.Now().Unix(),
	}

	// Payload has one data item
	post.Payload = append(post.Payload, data)

	// Encode to JSON
	jsonData, _ := json.MarshalIndent(&post, "", "\t")
	fmt.Printf("%s\n", jsonData)

	// Compress the JSON
	jsonCompressBuffer := bytes.Buffer{}

	jsonCompressWriter := gzip.NewWriter(&jsonCompressBuffer)
	_, _ = jsonCompressWriter.Write(jsonData)
	_ = jsonCompressWriter.Close()

	// Create a multipart form for upload
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition",
		fmt.Sprintf(`form-data; name="%s"; filename="%s"`,
			"data", "json.gz"))
	h.Set("Content-Type", "application/json")
	h.Set("Content-Encoding", "gzip")

	part, err := writer.CreatePart(h)
	_, _ = jsonCompressBuffer.WriteTo(part)
	_ = writer.Close()

	// Send it
	resp, err := http.Post("https://longview.linode.com/post", writer.FormDataContentType(), body)
	if err != nil {
		log.Printf("Cannot get http response data: %s", err)
		return 0, err
	}

	defer func() {
		if resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()

	// Parse response JSON
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Cannot get http response data: %s", err)
		return 0, err
	}

	sleepNew := 0

	if len(respBody) > 0 {
		var resp Response
		if json.Unmarshal(respBody, &resp) == nil {
			sleepNew = resp.Sleep
		}
	}

	return sleepNew, nil
}
