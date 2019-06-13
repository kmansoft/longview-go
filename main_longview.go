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

		sleepNew, err := sendDataToServer(apiKey, &data)
		if err != nil {
			sleep = 15
		} else if sleepNew > 0 {
			sleep = sleepNew
		}

		// Wait / sleep
		time.Sleep(time.Duration(sleep) * time.Second)
	}
}

func isApiKeyValid(apiKey string) bool {
	return len(apiKey) == 35
}

func sendDataToServer(apiKey string, data *Data) (int, error) {

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
	fmt.Printf("Output:\n%s\n", jsonData)

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
	req, err := http.NewRequest("POST", "https://longview.linode.com/post", body)
	if err != nil {
		log.Printf("Cannot make new http request: %s", err)
		return 0, err
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Cannot get http response data: %s", err)
		return 0, err
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	// Parse response - which may tell us for how long to sleep or to terminate
	fmt.Println("response Status:", resp.Status)
	fmt.Println("response Headers:", resp.Header)

	respBody, _ := ioutil.ReadAll(resp.Body)

	fmt.Println("response Body:", string(respBody))

	// Parse response JSON
	sleepNew := 0

	if len(respBody) > 0 {
		var resp Response
		if json.Unmarshal(respBody, &resp) == nil {
			sleepNew = resp.Sleep
		}
	}

	return sleepNew, nil
}
