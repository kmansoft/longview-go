package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"os"
	"strings"
	"time"
)

const (
	API_KEY_FILE    = "/etc/linode/longview.key"
	API_KEY_TEST    = "12345678-1234-1234-1223334445556677"
	CLIENT_VERSION  = "1.1.5"
	CONFIG_LOCATION = "/etc/linode/longview.d/"
	CONFIG_SUFFIX   = ".conf"
)

type Response struct {
	Die   string `json:"die"`
	Sleep int    `json:"sleep"`
}

func main() {
	// Read API key
	apiKey := ""
	apiKeyBytes, err := ioutil.ReadFile(API_KEY_FILE)
	if err == nil {
		for _, l := range strings.Split(strings.TrimSpace(string(apiKeyBytes)), "\n") {
			l = strings.TrimSpace(l)
			if strings.HasPrefix(l, "#") {
				continue
			}
			apiKey = l
			break
		}
	}
	if len(apiKey) == 0 {
		// No api key
		fmt.Printf("There is no API key, please set in %s\n", API_KEY_FILE)
		os.Exit(1)
	} else if !isApiKeyValid(apiKey) || isApiKeyTest(apiKey) {
		// Api key is set but not valid, or is our sample key
		fmt.Printf("The API key is not valid, please update in %s\n", API_KEY_FILE)
		os.Exit(1)
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
		_ = GetDataMemory(&data)
		_ = GetDataCPU(&data)

		_ = GetDataSysInfo(&data)

		_ = GetDataNetwork(&data)
		_ = GetDataDisks(&data)

		_ = GetDataProcessesPorts(&data)

		_ = GetDataAppNginx(client, &data)
		_ = GetDataAppApache(client, &data)
		_ = GetDataAppMysql(client, &data)

		// Send to server
		sleepNew, die, err := sendDataToServer(client, apiKey, &data)
		if err != nil {
			sleep = 15
		} else if sleepNew > 0 {
			sleep = sleepNew
		}

		// Server tells us to quit
		if die {
			fmt.Printf("Server told us to quit\n")
			break
		}

		// Wait / sleep
		fmt.Printf("Sleeping for %d seconds\n", sleep)
		time.Sleep(time.Duration(sleep) * time.Second)
	}
}

func isApiKeyValid(apiKey string) bool {
	return len(apiKey) == 35
}

func isApiKeyTest(apiKey string) bool {
	return apiKey == API_KEY_TEST
}

func sendDataToServer(client *http.Client, apiKey string, data *Data) (int, bool, error) {

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
	resp, err := client.Post("https://longview.linode.com/post", writer.FormDataContentType(), body)
	if err != nil {
		fmt.Printf("Cannot get http response data: %s\n", err)
		return 0, false, err
	}

	defer func() {
		if resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()

	// Parse response JSON
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Cannot get http response data: %s\n", err)
		return 0, false, err
	}

	sleepNew := 0
	die := false

	if len(respBody) > 0 {
		var resp Response
		if json.Unmarshal(respBody, &resp) == nil {
			die = resp.Die == "please"
			sleepNew = resp.Sleep
		}
	}

	return sleepNew, die, nil
}
